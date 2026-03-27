package tmux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"klein-harness/internal/dispatch"
)

func TestRunBoundedBurstWritesCommandBannerToLog(t *testing.T) {
	fakeBin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	tmuxRoot := filepath.Join(t.TempDir(), "tmux")
	if err := os.MkdirAll(tmuxRoot, 0o755); err != nil {
		t.Fatalf("mkdir fake tmux root: %v", err)
	}
	writeExecutable(t, filepath.Join(fakeBin, "tmux"), fakeTmuxForBurstTest)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_TMUX_ROOT", tmuxRoot)

	root := t.TempDir()
	checkpointPath := filepath.Join(root, "checkpoints", "task.json")
	outcomePath := filepath.Join(root, "checkpoints", "outcome.json")
	result, err := RunBoundedBurst(BurstRequest{
		Root:           root,
		TaskID:         "T-9",
		DispatchID:     "dispatch-9",
		WorkerID:       "worker-9",
		Cwd:            root,
		Command:        "printf 'hello\\n'",
		CommandBanner:  "[harness:T-9] planner-lane-check",
		CheckpointPath: checkpointPath,
		OutcomePath:    outcomePath,
		Budget: dispatch.Budget{
			MaxMinutes: 1,
		},
	})
	if err != nil {
		t.Fatalf("run bounded burst: %v", err)
	}
	if result.CommandBanner != "[harness:T-9] planner-lane-check" {
		t.Fatalf("commandBanner = %q", result.CommandBanner)
	}
	logPayload, err := os.ReadFile(result.LogPath)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	logText := string(logPayload)
	if !strings.Contains(logText, "[harness:T-9] planner-lane-check") {
		t.Fatalf("expected banner in log, got %q", logText)
	}
	if !strings.Contains(logText, "hello") {
		t.Fatalf("expected command output in log, got %q", logText)
	}
}
