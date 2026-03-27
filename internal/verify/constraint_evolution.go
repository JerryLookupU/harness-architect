package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"klein-harness/internal/adapter"
	"klein-harness/internal/orchestration"
	"klein-harness/internal/state"
)

const (
	evolutionRuleOwnedPathsNonEmptyAfterPathConflict = "evolution-owned-paths-nonempty-after-path-conflict"
)

type ConstraintEvolutionState struct {
	SchemaVersion   string         `json:"schemaVersion"`
	Generator       string         `json:"generator"`
	UpdatedAt       string         `json:"updatedAt"`
	TaskID          string         `json:"taskId"`
	CurrentHardRule []string       `json:"currentHardRuleIds,omitempty"`
	LatestFailure   map[string]int `json:"latestFailureCounts,omitempty"`
	LastDecision    string         `json:"lastDecision,omitempty"`
}

type ConstraintEvolutionAuditEvent struct {
	Timestamp     string         `json:"timestamp"`
	TaskID        string         `json:"taskId"`
	Action        string         `json:"action"`
	RuleID        string         `json:"ruleId"`
	FailureCounts map[string]int `json:"failureCounts,omitempty"`
	Reason        string         `json:"reason,omitempty"`
}

func constraintEvolutionStatePath(root, taskID string) (string, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.StateDir, fmt.Sprintf("constraint-evolution-%s.json", taskID)), nil
}

func constraintEvolutionAuditLogPath(root string) (string, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.StateDir, "constraint-evolution-audit.jsonl"), nil
}

func appendAuditEvent(path string, ev ConstraintEvolutionAuditEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(payload, '\n'))
	return err
}

// EvolveConstraintSystemFromFeedback implements failure-driven automatic tightening.
// Current MVP: repeated path_conflict (>=2 in RecentFailures) promotes the "owned paths non-empty" gate.
func EvolveConstraintSystemFromFeedback(root string, task adapter.Task, base orchestration.ConstraintSystem) (orchestration.ConstraintSystem, error) {
	evolved := base

	// Load feedback memory (evidence-first).
	summary, err := LoadFeedbackSummary(root)
	if err != nil {
		return base, err
	}
	taskSummary, ok := CurrentTaskFeedback(summary, task.TaskID)
	_ = ok

	failureCounts := map[string]int{}
	if ok {
		for _, ev := range taskSummary.RecentFailures {
			if ev.FeedbackType == "" {
				continue
			}
			failureCounts[ev.FeedbackType]++
		}
	}

	// Decision policy (failure-driven automatic tightening).
	pathConflictCount := failureCounts["path_conflict"]
	shouldPromote := pathConflictCount >= 2

	// Apply enforcement promotion to the target rule.
	for i := range evolved.Rules {
		if evolved.Rules[i].ID != evolutionRuleOwnedPathsNonEmptyAfterPathConflict {
			continue
		}
		if shouldPromote {
			evolved.Rules[i].Enforcement = "hard"
		} else {
			evolved.Rules[i].Enforcement = "soft"
		}
	}

	// Audit + rollback bookkeeping (explainable, replayable).
	statePath, err := constraintEvolutionStatePath(root, task.TaskID)
	if err != nil {
		return base, err
	}
	var prev ConstraintEvolutionState
	_ = json.Unmarshal(mustReadBytesIfExists(statePath), &prev) // best-effort

	newHard := []string{}
	if shouldPromote {
		newHard = []string{evolutionRuleOwnedPathsNonEmptyAfterPathConflict}
	}

	// Write audit only when the decision changed.
	prevPromoted := len(prev.CurrentHardRule) > 0
	if shouldPromote != prevPromoted {
		action := "promote"
		if !shouldPromote {
			action = "rollback"
		}
		ev := ConstraintEvolutionAuditEvent{
			Timestamp:     state.NowUTC(),
			TaskID:        task.TaskID,
			Action:        action,
			RuleID:        evolutionRuleOwnedPathsNonEmptyAfterPathConflict,
			FailureCounts: failureCounts,
			Reason:        fmt.Sprintf("path_conflict_count=%d threshold=%d", pathConflictCount, 2),
		}
		auditPath, auditErr := constraintEvolutionAuditLogPath(root)
		if auditErr == nil {
			_ = appendAuditEvent(auditPath, ev)
		}
	}

	newState := ConstraintEvolutionState{
		SchemaVersion:   "kh.constraint-evolution.v1",
		Generator:       "kh-runtime",
		UpdatedAt:       state.NowUTC(),
		TaskID:          task.TaskID,
		CurrentHardRule: newHard,
		LatestFailure:   failureCounts,
		LastDecision:    fmt.Sprintf("path_conflict=%d shouldPromote=%t", pathConflictCount, shouldPromote),
	}
	if payload, err := json.MarshalIndent(newState, "", "  "); err == nil {
		_ = os.WriteFile(statePath, append(payload, '\n'), 0o644)
	}

	return evolved, nil
}

// mustReadBytesIfExists returns file bytes if exists, otherwise an empty slice.
func mustReadBytesIfExists(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return []byte{}
	}
	return b
}
