package worktree

import "testing"

func TestPathOverlap(t *testing.T) {
	if !PathOverlap("internal/worker/**", "internal/worker/main.go") {
		t.Fatalf("expected overlap")
	}
	if PathOverlap("docs/runtime", "internal/worker/main.go") {
		t.Fatalf("unexpected overlap")
	}
}

func TestGuardArtifacts(t *testing.T) {
	if err := GuardArtifacts([]string{"docs/runtime/**"}, []string{"docs/runtime/a.md"}); err != nil {
		t.Fatalf("guard artifacts: %v", err)
	}
	if err := GuardArtifacts([]string{"docs/runtime/**"}, []string{"internal/worker/main.go"}); err == nil {
		t.Fatalf("expected artifact guard failure")
	}
}
