package runtime

import (
	"fmt"
	"strings"

	"klein-harness/internal/adapter"
	"klein-harness/internal/state"
	"klein-harness/internal/tmux"
	"klein-harness/internal/verify"
)

func FinalizeTaskAfterVerification(root, taskID, dispatchID, verifyStatus, verifySummary, verifyPath, followUp string, verifyErr error) (adapter.Task, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return adapter.Task{}, err
	}
	now := state.NowUTC()
	runtimeFollowUp := followUp
	analysisLoop := shouldEnterAnalysisLoop("", verifyStatus, followUp, verifyErr)
	if analysisLoop {
		runtimeFollowUp = "analysis.required"
	}
	taskStatus := ""
	switch {
	case verifyStatus == "passed" && verifyErr == nil && followUp == "task.completed":
		taskStatus = "completed"
	case analysisLoop:
		taskStatus = "needs_replan"
	case verifyErr != nil:
		taskStatus = "blocked"
	case verifyStatus == "blocked":
		taskStatus = "blocked"
	default:
		taskStatus = "queued"
	}
	verifyCompleted := followUp == "task.completed" && verifyErr == nil
	if err := updateTask(root, taskID, func(current *adapter.Task) {
		current.Status = taskStatus
		current.StatusReason = coalesce(runtimeFollowUp, verifySummary, errorString(verifyErr))
		current.LastDispatchID = coalesce(dispatchID, current.LastDispatchID)
		current.LastLeaseID = ""
		current.VerificationStatus = verifyStatus
		current.VerificationSummary = verifySummary
		current.VerificationResultPath = verifyPath
		if analysisLoop {
			current.PlanEpoch++
			current.PromptStages = analysisPromptStages()
			current.ResumeStrategy = "fresh"
			current.PreferredResumeSessionID = ""
			current.CandidateResumeSessionIDs = nil
			current.CompletedAt = ""
		}
		if verifyCompleted {
			current.CompletedAt = now
		}
		current.UpdatedAt = now
	}); err != nil {
		return adapter.Task{}, err
	}
	if err := updateVerification(paths.VerificationSummaryPath, VerificationEntry{
		TaskID:     taskID,
		DispatchID: dispatchID,
		Status:     verifyStatus,
		Summary:    verifySummary,
		ResultPath: verifyPath,
		UpdatedAt:  now,
		Completed:  verifyCompleted,
		FollowUp:   coalesce(runtimeFollowUp, errorString(verifyErr)),
	}); err != nil {
		return adapter.Task{}, err
	}
	if err := updateRuntime(paths.RuntimePath, func(current RuntimeState) RuntimeState {
		current.Status = taskStatus
		current.ActiveTaskID = taskID
		current.LastRunAt = now
		current.LastError = errorString(verifyErr)
		return current
	}); err != nil {
		return adapter.Task{}, err
	}
	task, err := adapter.LoadTask(root, taskID)
	if err != nil {
		return adapter.Task{}, err
	}
	if err := refreshExecutionIndexes(paths, task, "", ""); err != nil {
		return adapter.Task{}, err
	}
	return task, nil
}

func RestartFromStage(root, taskID, stage, reason string) (adapter.Task, error) {
	if strings.TrimSpace(stage) == "" {
		stage = "queued"
	}
	task, err := adapter.LoadTask(root, taskID)
	if err != nil {
		return adapter.Task{}, err
	}
	if sessionName := taskTmuxSession(root, task); sessionName != "" {
		_ = tmux.KillSession(sessionName)
	}
	if err := updateTask(root, taskID, func(current *adapter.Task) {
		current.Status = "queued"
		current.StatusReason = coalesce(reason, "restarted from "+stage)
		current.LastLeaseID = ""
		current.VerificationStatus = ""
		current.VerificationSummary = ""
		current.VerificationResultPath = ""
		current.CompletedAt = ""
		current.ArchivedAt = ""
		current.UpdatedAt = state.NowUTC()
	}); err != nil {
		return adapter.Task{}, err
	}
	return adapter.LoadTask(root, taskID)
}

func StopTask(root, taskID, reason string) (adapter.Task, error) {
	task, err := adapter.LoadTask(root, taskID)
	if err != nil {
		return adapter.Task{}, err
	}
	if sessionName := taskTmuxSession(root, task); sessionName != "" {
		_ = tmux.KillSession(sessionName)
	}
	if err := updateTask(root, taskID, func(current *adapter.Task) {
		current.Status = "blocked"
		current.StatusReason = coalesce(reason, "stopped by operator")
		current.LastLeaseID = ""
		current.UpdatedAt = state.NowUTC()
	}); err != nil {
		return adapter.Task{}, err
	}
	return adapter.LoadTask(root, taskID)
}

func ArchiveTask(root, taskID, reason string) (adapter.Task, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return adapter.Task{}, err
	}
	task, err := adapter.LoadTask(root, taskID)
	if err != nil {
		return adapter.Task{}, err
	}
	var gate verify.CompletionGate
	if ok, err := state.LoadJSONIfExists(paths.CompletionGatePath, &gate); err != nil {
		return adapter.Task{}, err
	} else if !ok || gate.TaskID != taskID || !gate.Satisfied || gate.Retired {
		return adapter.Task{}, fmt.Errorf("%w: task=%s", verify.ErrCompletionGateOpen, taskID)
	}
	var guard verify.GuardState
	_, _ = state.LoadJSONIfExists(paths.GuardStatePath, &guard)
	if sessionName := taskTmuxSession(root, task); sessionName != "" {
		_ = tmux.KillSession(sessionName)
	}
	gate.Retired = true
	gate.Status = "retired"
	gate.RetireEligible = false
	if _, err := state.WriteSnapshot(paths.CompletionGatePath, &gate, "harness-control", gate.Revision); err != nil {
		return adapter.Task{}, err
	}
	guard.Status = "archived"
	guard.TaskID = taskID
	guard.DispatchID = gate.DispatchID
	guard.SafeToArchive = false
	guard.CompletionGateStatus = gate.Status
	guard.CompletionGateSatisfied = gate.Satisfied
	guard.RetireEligible = false
	if _, err := state.WriteSnapshot(paths.GuardStatePath, &guard, "harness-control", guard.Revision); err != nil {
		return adapter.Task{}, err
	}
	if err := updateTask(root, taskID, func(current *adapter.Task) {
		current.Status = "archived"
		current.StatusReason = coalesce(reason, "archived")
		current.ArchivedAt = state.NowUTC()
		current.UpdatedAt = state.NowUTC()
	}); err != nil {
		return adapter.Task{}, err
	}
	return adapter.LoadTask(root, taskID)
}

func taskTmuxSession(root string, task adapter.Task) string {
	if task.TmuxSession != "" {
		return task.TmuxSession
	}
	session, ok, err := tmux.FindTaskSession(root, task.TaskID, "")
	if err != nil || !ok {
		return ""
	}
	return session.SessionName
}
