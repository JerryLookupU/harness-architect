package verify

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"klein-harness/internal/adapter"
	"klein-harness/internal/bootstrap"
	"klein-harness/internal/orchestration"
	"klein-harness/internal/state"
)

func writeFeedbackSummaryForPathConflict(t *testing.T, root string, taskID string, count int) {
	t.Helper()
	paths, err := adapter.Resolve(root)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	recent := make([]FeedbackEvent, 0, count)
	for i := 0; i < count; i++ {
		recent = append(recent, FeedbackEvent{
			ID:           "FB-" + filepath.Base(filepath.Join(root, "x")), // stable-ish; not used for logic
			TaskID:       taskID,
			FeedbackType: "path_conflict",
			Severity:     "critical",
			Source:       "verification",
			Step:         "verify",
			Message:      "owned path violation",
			Timestamp:    state.NowUTC(),
		})
	}

	summary := FeedbackSummary{
		SchemaVersion:       "kh.feedback-summary.v1",
		Generator:           "test",
		GeneratedAt:         state.NowUTC(),
		FeedbackLogPath:     ".harness/feedback-log.jsonl",
		FeedbackEventCount:  count,
		ErrorCount:          count,
		CriticalCount:       count,
		IllegalActionCount:  0,
		TaskFeedbackSummary: map[string]TaskFeedbackSummary{},
		ByType:              map[string]int{"path_conflict": count},
		BySeverity:          map[string]int{"critical": count},
		RecentFailures:      recent,
	}
	summary.TaskFeedbackSummary[taskID] = TaskFeedbackSummary{
		TaskID:             taskID,
		FeedbackCount:      count,
		ErrorCount:         count,
		CriticalCount:      count,
		LatestFeedbackType: "path_conflict",
		LatestSeverity:     "critical",
		LatestMessage:      "owned path violation",
		LatestTimestamp:    state.NowUTC(),
		RecentFailures:     recent,
	}

	payloadPath := filepath.Join(paths.StateDir, "feedback-summary.json")
	b, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("marshal feedback summary: %v", err)
	}
	if err := os.WriteFile(payloadPath, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("write feedback-summary: %v", err)
	}
}

func TestEvolveConstraintSystemFromFeedbackPromotesPathConflictGate(t *testing.T) {
	root := t.TempDir()
	if _, err := bootstrap.Init(root); err != nil {
		t.Fatalf("init: %v", err)
	}

	writeFeedbackSummaryForPathConflict(t, root, "T-1", 2)

	base := orchestration.DefaultConstraintSystem(root, []string{"dispatch_ready"})
	evolved, err := EvolveConstraintSystemFromFeedback(root, adapter.Task{TaskID: "T-1"}, base)
	if err != nil {
		t.Fatalf("evolve: %v", err)
	}

	var rule orchestration.ConstraintRule
	found := false
	for _, r := range evolved.Rules {
		if r.ID == evolutionRuleOwnedPathsNonEmptyAfterPathConflict {
			rule = r
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("evolution rule not found in evolved constraint system")
	}
	if rule.Enforcement != "hard" {
		t.Fatalf("expected evolution rule enforcement=hard, got %s", rule.Enforcement)
	}
}

func TestEvolveConstraintSystemFromFeedbackRollsBackWhenPathConflictDrops(t *testing.T) {
	root := t.TempDir()
	if _, err := bootstrap.Init(root); err != nil {
		t.Fatalf("init: %v", err)
	}

	writeFeedbackSummaryForPathConflict(t, root, "T-1", 1)
	base := orchestration.DefaultConstraintSystem(root, []string{"dispatch_ready"})
	evolved, err := EvolveConstraintSystemFromFeedback(root, adapter.Task{TaskID: "T-1"}, base)
	if err != nil {
		t.Fatalf("evolve: %v", err)
	}

	for _, r := range evolved.Rules {
		if r.ID == evolutionRuleOwnedPathsNonEmptyAfterPathConflict {
			if r.Enforcement != "soft" {
				t.Fatalf("expected evolution rule enforcement=soft, got %s", r.Enforcement)
			}
			return
		}
	}
	t.Fatalf("evolution rule not found in evolved constraint system")
}

func TestOwnedPathNonEmptyEvolutionGateBlocksCompletionWhenChangedPathsEmpty(t *testing.T) {
	root := t.TempDir()
	ticket := issueTestDispatch(t, root)

	relVerifyPath := writeVerificationArtifacts(t, root, ticket.DispatchID, false)
	// overwrite worker-result.json with empty changedPaths
	paths, err := adapter.Resolve(root)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	artifactDir := filepath.Join(paths.ArtifactsDir, "T-1", ticket.DispatchID)
	if err := os.WriteFile(filepath.Join(artifactDir, "worker-result.json"), []byte(`{"status":"succeeded","changedPaths":[]}`), 0o644); err != nil {
		t.Fatalf("overwrite worker-result: %v", err)
	}

	// ensure task has ownedPaths for the gate to evaluate
	upsertTask(t, root, adapter.Task{
		TaskID:     "T-1",
		ThreadKey:  "thread-1",
		PlanEpoch:  1,
		OwnedPaths: []string{"internal/verify/**"},
	})

	// write promoted (hard) constraint snapshot
	system := orchestration.DefaultConstraintSystem(root, []string{"dispatch_ready"})
	for i := range system.Rules {
		if system.Rules[i].ID == evolutionRuleOwnedPathsNonEmptyAfterPathConflict {
			system.Rules[i].Enforcement = "hard"
		}
	}
	softRules, hardRules := orchestration.SplitConstraintRules(system)
	if err := orchestration.WriteConstraintSnapshot(orchestration.ConstraintSnapshotPath(root, "T-1"), orchestration.ConstraintSnapshot{
		SchemaVersion:    "kh.constraint-snapshot.v1",
		Generator:        "test",
		GeneratedAt:      "2026-03-26T10:00:00Z",
		TaskID:           "T-1",
		DispatchID:       ticket.DispatchID,
		PlanEpoch:        1,
		ConstraintSystem: system,
		SoftRules:        softRules,
		HardRules:        hardRules,
	}); err != nil {
		t.Fatalf("write constraint snapshot: %v", err)
	}

	writeAcceptedPacketAndContract(t, root, ticket)

	_, err = Ingest(Request{
		Root:                   root,
		TaskID:                 "T-1",
		DispatchID:             ticket.DispatchID,
		PlanEpoch:              1,
		Attempt:                1,
		CausationID:            "outcome-1",
		Status:                 "passed",
		Summary:                "verification passed but evolution gate blocks completion",
		VerificationResultPath: relVerifyPath,
	})
	if err == nil || !errors.Is(err, ErrCompletionGateOpen) {
		t.Fatalf("expected completion gate open error, got %v", err)
	}

	gate := loadCompletionGate(t, root)
	if gate.Satisfied {
		t.Fatalf("expected completion gate to be unsatisfied due to owned_path_nonempty evolution gate")
	}
	found := false
	for _, hc := range gate.HardConstraintChecks {
		if hc.Name == evolutionRuleOwnedPathsNonEmptyAfterPathConflict {
			found = true
			if hc.OK {
				t.Fatalf("expected hard constraint check to fail, got %+v", hc)
			}
		}
	}
	if !found {
		t.Fatalf("expected hard constraint check for evolution rule not found in gate.HardConstraintChecks")
	}
}
