package runtime

import (
	"errors"
	"path/filepath"
	"testing"

	"klein-harness/internal/adapter"
	"klein-harness/internal/bootstrap"
	"klein-harness/internal/state"
	"klein-harness/internal/verify"
)

func TestRestartFromStageResetsTaskToQueued(t *testing.T) {
	root := t.TempDir()
	if _, err := bootstrap.Init(root); err != nil {
		t.Fatalf("init: %v", err)
	}
	task := adapter.Task{
		TaskID:                 "T-001",
		ThreadKey:              "R-001",
		Kind:                   "feature",
		Title:                  "test",
		Summary:                "test",
		Status:                 "completed",
		VerificationStatus:     "passed",
		VerificationSummary:    "ok",
		VerificationResultPath: "artifact/verify.json",
		CompletedAt:            state.NowUTC(),
	}
	if err := adapter.UpsertTask(root, task); err != nil {
		t.Fatalf("upsert task: %v", err)
	}
	updated, err := RestartFromStage(root, task.TaskID, "queued", "")
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if updated.Status != "queued" {
		t.Fatalf("expected queued status, got %q", updated.Status)
	}
	if updated.VerificationStatus != "" || updated.CompletedAt != "" {
		t.Fatalf("expected verification fields to clear: %#v", updated)
	}
}

func TestArchiveTaskRequiresSatisfiedCompletionGate(t *testing.T) {
	root := t.TempDir()
	paths, err := bootstrap.Init(root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	task := adapter.Task{
		TaskID:    "T-001",
		ThreadKey: "R-001",
		Kind:      "feature",
		Title:     "test",
		Summary:   "test",
		Status:    "completed",
	}
	if err := adapter.UpsertTask(root, task); err != nil {
		t.Fatalf("upsert task: %v", err)
	}
	if _, err := ArchiveTask(root, task.TaskID, ""); err == nil {
		t.Fatalf("expected archive to fail without gate")
	}
	gate := verify.CompletionGate{
		Status:     "satisfied",
		Satisfied:  true,
		TaskID:     task.TaskID,
		DispatchID: "dispatch_T_001_1_1",
	}
	gateRevision, err := state.CurrentRevision(paths.CompletionGatePath)
	if err != nil {
		t.Fatalf("gate revision: %v", err)
	}
	if _, err := state.WriteSnapshot(paths.CompletionGatePath, &gate, "test", gateRevision); err != nil {
		t.Fatalf("write gate: %v", err)
	}
	guard := verify.GuardState{
		Status:                  "retire_ready",
		TaskID:                  task.TaskID,
		CompletionGateStatus:    "satisfied",
		CompletionGateSatisfied: true,
		RetireEligible:          true,
		SafeToArchive:           true,
	}
	guardRevision, err := state.CurrentRevision(paths.GuardStatePath)
	if err != nil {
		t.Fatalf("guard revision: %v", err)
	}
	if _, err := state.WriteSnapshot(paths.GuardStatePath, &guard, "test", guardRevision); err != nil {
		t.Fatalf("write guard: %v", err)
	}
	updated, err := ArchiveTask(root, task.TaskID, "")
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if updated.Status != "archived" {
		t.Fatalf("expected archived status, got %q", updated.Status)
	}
	var archivedGate verify.CompletionGate
	if err := state.LoadJSON(paths.CompletionGatePath, &archivedGate); err != nil {
		t.Fatalf("load gate: %v", err)
	}
	if !archivedGate.Retired {
		t.Fatalf("expected retired gate: %#v", archivedGate)
	}
}

func TestArchiveTaskReadsTaskScopedGate(t *testing.T) {
	root := t.TempDir()
	paths, err := bootstrap.Init(root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	task := adapter.Task{
		TaskID:    "T-002",
		ThreadKey: "R-002",
		Kind:      "feature",
		Title:     "task-scoped archive",
		Summary:   "task-scoped archive",
		Status:    "completed",
	}
	if err := adapter.UpsertTask(root, task); err != nil {
		t.Fatalf("upsert task: %v", err)
	}
	if _, err := state.WriteSnapshot(paths.CompletionGateTaskPath(task.TaskID), &verify.CompletionGate{
		Status:     "satisfied",
		Satisfied:  true,
		TaskID:     task.TaskID,
		DispatchID: "dispatch_T_002_1_1",
	}, "test", 0); err != nil {
		t.Fatalf("write task-scoped gate: %v", err)
	}
	if _, err := state.WriteSnapshot(paths.GuardStateTaskPath(task.TaskID), &verify.GuardState{
		Status:                  "retire_ready",
		TaskID:                  task.TaskID,
		CompletionGateStatus:    "satisfied",
		CompletionGateSatisfied: true,
		RetireEligible:          true,
		SafeToArchive:           true,
	}, "test", 0); err != nil {
		t.Fatalf("write task-scoped guard: %v", err)
	}

	updated, err := ArchiveTask(root, task.TaskID, "")
	if err != nil {
		t.Fatalf("archive from task-scoped gate: %v", err)
	}
	if updated.Status != "archived" {
		t.Fatalf("expected archived status, got %q", updated.Status)
	}
	var archivedGate verify.CompletionGate
	if err := state.LoadJSON(paths.CompletionGateTaskPath(task.TaskID), &archivedGate); err != nil {
		t.Fatalf("load task-scoped gate: %v", err)
	}
	if !archivedGate.Retired {
		t.Fatalf("expected task-scoped gate to be retired: %#v", archivedGate)
	}
}

func TestFinalizeTaskAfterVerificationCompletesTask(t *testing.T) {
	root := t.TempDir()
	paths, err := bootstrap.Init(root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	task := adapter.Task{
		TaskID:    "T-010",
		ThreadKey: "thread-10",
		Kind:      "feature",
		Title:     "complete",
		Summary:   "complete",
		Status:    "running",
	}
	if err := adapter.UpsertTask(root, task); err != nil {
		t.Fatalf("upsert task: %v", err)
	}

	updated, err := FinalizeTaskAfterVerification(root, task.TaskID, "dispatch-10", "passed", "verified", filepath.Join(root, "verify.json"), "task.completed", nil)
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if updated.Status != "completed" || updated.VerificationStatus != "passed" || updated.CompletedAt == "" {
		t.Fatalf("expected completed task after finalize: %#v", updated)
	}
	var verificationSummary VerificationSummary
	if err := state.LoadJSON(paths.VerificationSummaryPath, &verificationSummary); err != nil {
		t.Fatalf("load verification summary: %v", err)
	}
	if verificationSummary.Tasks[task.TaskID].FollowUp != "task.completed" {
		t.Fatalf("unexpected verification summary entry: %+v", verificationSummary.Tasks[task.TaskID])
	}
}

func TestFinalizeTaskAfterVerificationMovesTaskToNeedsReplan(t *testing.T) {
	root := t.TempDir()
	if _, err := bootstrap.Init(root); err != nil {
		t.Fatalf("init: %v", err)
	}
	task := adapter.Task{
		TaskID:       "T-011",
		ThreadKey:    "thread-11",
		Kind:         "feature",
		Title:        "replan",
		Summary:      "replan",
		Status:       "running",
		PlanEpoch:    2,
		PromptStages: []string{"route", "dispatch"},
	}
	if err := adapter.UpsertTask(root, task); err != nil {
		t.Fatalf("upsert task: %v", err)
	}

	updated, err := FinalizeTaskAfterVerification(root, task.TaskID, "dispatch-11", "failed", "needs more work", filepath.Join(root, "verify.json"), "replan.emitted", nil)
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if updated.Status != "needs_replan" || updated.PlanEpoch != 3 {
		t.Fatalf("expected needs_replan with incremented epoch: %#v", updated)
	}
	if len(updated.PromptStages) == 0 || updated.PromptStages[0] != "analysis" {
		t.Fatalf("expected analysis prompt stages after replan: %#v", updated.PromptStages)
	}
}

func TestFinalizeTaskAfterVerificationBlocksOnVerificationError(t *testing.T) {
	root := t.TempDir()
	if _, err := bootstrap.Init(root); err != nil {
		t.Fatalf("init: %v", err)
	}
	task := adapter.Task{
		TaskID:    "T-012",
		ThreadKey: "thread-12",
		Kind:      "feature",
		Title:     "blocked",
		Summary:   "blocked",
		Status:    "running",
	}
	if err := adapter.UpsertTask(root, task); err != nil {
		t.Fatalf("upsert task: %v", err)
	}

	updated, err := FinalizeTaskAfterVerification(root, task.TaskID, "dispatch-12", "passed", "gate open", filepath.Join(root, "verify.json"), "", errors.New("completion gate is not satisfied"))
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if updated.Status != "blocked" {
		t.Fatalf("expected blocked task after verification error: %#v", updated)
	}
}
