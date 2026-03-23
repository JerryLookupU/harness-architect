package route

import "klein-harness/internal/worktree"

type Input struct {
	TaskID                   string
	RoleHint                 string
	Kind                     string
	WorkerMode               string
	PlanEpoch                int
	LatestPlanEpoch          int
	ResumeStrategy           string
	PreferredResumeSessionID string
	CandidateResumeSessionIDs []string
	SessionContested         bool
	CheckpointRequired       bool
	CheckpointFresh          bool
	WorktreePath             string
	OwnedPaths               []string
	RequiredSummaryVersion   string
}

type Decision struct {
	Route                  string   `json:"route"`
	DispatchReady          bool     `json:"dispatchReady"`
	ReasonCodes            []string `json:"reasonCodes"`
	RequiredSummaryVersion string   `json:"requiredSummaryVersion"`
	ResumeSessionID        string   `json:"resumeSessionId,omitempty"`
	WorktreePath           string   `json:"worktreePath,omitempty"`
	OwnedPaths             []string `json:"ownedPaths,omitempty"`
}

func Evaluate(input Input) Decision {
	reasons := make([]string, 0)
	if input.LatestPlanEpoch > 0 && input.PlanEpoch > 0 && input.PlanEpoch < input.LatestPlanEpoch {
		return Decision{
			Route:                  "replan",
			DispatchReady:          false,
			ReasonCodes:            []string{"plan_epoch_stale"},
			RequiredSummaryVersion: input.RequiredSummaryVersion,
			WorktreePath:           input.WorktreePath,
			OwnedPaths:             input.OwnedPaths,
		}
	}
	if input.CheckpointRequired {
		return Decision{
			Route:                  "block",
			DispatchReady:          false,
			ReasonCodes:            []string{"checkpoint_required"},
			RequiredSummaryVersion: input.RequiredSummaryVersion,
			WorktreePath:           input.WorktreePath,
			OwnedPaths:             input.OwnedPaths,
		}
	}
	if worktree.RequiresIsolatedWorktree(input.RoleHint, input.Kind, input.WorkerMode) {
		if input.WorktreePath == "" {
			return Decision{
				Route:                  "block",
				DispatchReady:          false,
				ReasonCodes:            []string{"worktree_missing"},
				RequiredSummaryVersion: input.RequiredSummaryVersion,
			}
		}
		if len(input.OwnedPaths) == 0 {
			return Decision{
				Route:                  "block",
				DispatchReady:          false,
				ReasonCodes:            []string{"owned_paths_missing"},
				RequiredSummaryVersion: input.RequiredSummaryVersion,
				WorktreePath:           input.WorktreePath,
			}
		}
	}

	if input.ResumeStrategy == "resume" || input.PreferredResumeSessionID != "" || len(input.CandidateResumeSessionIDs) > 0 {
		if input.SessionContested {
			return Decision{
				Route:                  "block",
				DispatchReady:          false,
				ReasonCodes:            []string{"resume_session_contested"},
				RequiredSummaryVersion: input.RequiredSummaryVersion,
				WorktreePath:           input.WorktreePath,
				OwnedPaths:             input.OwnedPaths,
			}
		}
		if input.CheckpointFresh && input.PreferredResumeSessionID != "" {
			return Decision{
				Route:                  "resume",
				DispatchReady:          true,
				ReasonCodes:            []string{"checkpoint_fresh", "owned_paths_valid"},
				RequiredSummaryVersion: input.RequiredSummaryVersion,
				ResumeSessionID:        input.PreferredResumeSessionID,
				WorktreePath:           input.WorktreePath,
				OwnedPaths:             input.OwnedPaths,
			}
		}
		reasons = append(reasons, "checkpoint_stale_fresh_start")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "dispatch_ready")
	}
	return Decision{
		Route:                  "dispatch",
		DispatchReady:          true,
		ReasonCodes:            reasons,
		RequiredSummaryVersion: input.RequiredSummaryVersion,
		WorktreePath:           input.WorktreePath,
		OwnedPaths:             input.OwnedPaths,
	}
}
