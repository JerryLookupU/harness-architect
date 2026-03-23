package tmux

import (
	"path/filepath"
	"testing"

	"klein-harness/internal/dispatch"
)

func TestRunBoundedBurstWritesOutcome(t *testing.T) {
	root := t.TempDir()
	checkpointPath := filepath.Join(root, "checkpoints", "task.json")
	outcomePath := filepath.Join(root, "checkpoints", "outcome.json")
	result, err := RunBoundedBurst(BurstRequest{
		TaskID:         "T-1",
		DispatchID:     "dispatch-1",
		WorkerID:       "worker-1",
		Cwd:            root,
		Command:        "printf hello",
		CheckpointPath: checkpointPath,
		OutcomePath:    outcomePath,
		Budget: dispatch.Budget{
			MaxMinutes: 1,
		},
	})
	if err != nil {
		t.Fatalf("run bounded burst: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
}
