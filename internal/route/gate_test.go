package route

import "testing"

func TestEvaluateResumeDecision(t *testing.T) {
	decision := Evaluate(Input{
		TaskID:                   "T-1",
		RoleHint:                 "worker",
		Kind:                     "feature",
		PlanEpoch:                3,
		LatestPlanEpoch:          3,
		ResumeStrategy:           "resume",
		PreferredResumeSessionID: "sess-1",
		CheckpointFresh:          true,
		WorktreePath:             ".worktrees/T-1",
		OwnedPaths:               []string{"internal/worker/**"},
		RequiredSummaryVersion:   "state.v1",
	})
	if decision.Route != "resume" || !decision.DispatchReady {
		t.Fatalf("expected resumable decision, got %+v", decision)
	}
}

func TestEvaluateBlocksMissingWorktree(t *testing.T) {
	decision := Evaluate(Input{
		TaskID:                 "T-1",
		RoleHint:               "worker",
		Kind:                   "feature",
		PlanEpoch:              1,
		LatestPlanEpoch:        1,
		RequiredSummaryVersion: "state.v1",
	})
	if decision.Route != "block" {
		t.Fatalf("expected blocked route, got %+v", decision)
	}
}
