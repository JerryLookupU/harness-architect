package route

import "testing"

func TestEvaluateResumeDecision(t *testing.T) {
	decision := Evaluate(Input{
		TaskID:                   "T-1",
		RoleHint:                 "worker",
		Kind:                     "feature",
		Title:                    "Continue the previous implementation",
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
	for _, want := range []string{
		"checkpoint_fresh",
		"owned_paths_valid",
		"policy_resume_state_first",
		"policy_verify_evidence_required",
		"policy_review_if_multi_file_or_high_risk",
	} {
		if !containsReason(decision.ReasonCodes, want) {
			t.Fatalf("resume decision missing %q: %+v", want, decision.ReasonCodes)
		}
	}
}

func TestEvaluateBlocksMissingWorktree(t *testing.T) {
	decision := Evaluate(Input{
		TaskID:                 "T-1",
		RoleHint:               "worker",
		Kind:                   "feature",
		Title:                  "Implement the feature",
		PlanEpoch:              1,
		LatestPlanEpoch:        1,
		RequiredSummaryVersion: "state.v1",
	})
	if decision.Route != "block" {
		t.Fatalf("expected blocked route, got %+v", decision)
	}
}

func TestEvaluateBugRequestAddsDebuggingPolicy(t *testing.T) {
	decision := Evaluate(Input{
		TaskID:                 "T-bug",
		RoleHint:               "worker",
		Kind:                   "bug",
		Title:                  "Fix regression after verify failure",
		Summary:                "Unexpected error in route dispatch",
		PlanEpoch:              1,
		LatestPlanEpoch:        1,
		WorktreePath:           ".worktrees/T-bug",
		OwnedPaths:             []string{"internal/route/**"},
		RequiredSummaryVersion: "state.v1",
	})
	for _, want := range []string{
		"policy_bug_rca_first",
		"policy_verify_evidence_required",
		"policy_review_if_multi_file_or_high_risk",
	} {
		if !containsReason(decision.ReasonCodes, want) {
			t.Fatalf("bug decision missing %q: %+v", want, decision.ReasonCodes)
		}
	}
}

func TestEvaluateRecommendationAddsOptionsPolicy(t *testing.T) {
	decision := Evaluate(Input{
		TaskID:                 "T-design",
		RoleHint:               "worker",
		Kind:                   "design",
		Title:                  "Recommend the best way to route review tasks",
		Summary:                "Compare options and tradeoffs",
		PlanEpoch:              1,
		LatestPlanEpoch:        1,
		WorktreePath:           ".worktrees/T-design",
		OwnedPaths:             []string{"prompts/spec/**"},
		RequiredSummaryVersion: "state.v1",
	})
	if !containsReason(decision.ReasonCodes, "policy_options_before_plan") {
		t.Fatalf("recommendation decision missing options policy: %+v", decision.ReasonCodes)
	}
}

func containsReason(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
