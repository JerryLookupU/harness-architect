package verify

import (
	"errors"
	"fmt"

	"klein-harness/internal/a2a"
	"klein-harness/internal/adapter"
	"klein-harness/internal/dispatch"
	"klein-harness/internal/rca"
)

var ErrNoopWithoutEvidence = errors.New("noop completion requires verified acceptance evidence")
var ErrCompletionGateOpen = errors.New("completion gate is not satisfied")
var ErrVerifiedWithoutEvidence = errors.New("verified completion requires evidence")
var ErrReviewEvidenceRequired = errors.New("review-required task is missing review evidence")
var ErrAcceptedPacketRequired = errors.New("accepted packet is required before completion")
var ErrTaskContractRequired = errors.New("task contract is required before completion")
var ErrVerificationScorecardRequired = errors.New("verification scorecard is required before completion")
var ErrEvidenceLedgerRequired = errors.New("evidence ledger is required before completion")
var ErrBlockingVerificationFindings = errors.New("blocking verification findings must be cleared before completion")
var ErrTaskContractIncomplete = errors.New("task contract definition is incomplete")
var ErrExecutionTasksRemaining = errors.New("accepted packet still has remaining execution slices")

type Request struct {
	Root                   string
	RequestID              string
	TaskID                 string
	DispatchID             string
	PlanEpoch              int
	Attempt                int
	CausationID            string
	ReasonCodes            []string
	Status                 string
	Summary                string
	VerificationResultPath string
	FollowUp               string
}

type Result struct {
	VerificationEvent string `json:"verificationEvent"`
	FollowUpEvent     string `json:"followUpEvent,omitempty"`
}

func Ingest(request Request) (Result, error) {
	paths, err := adapter.Resolve(request.Root)
	if err != nil {
		return Result{}, err
	}
	var ticket dispatch.Ticket
	ticketFound := false
	if request.DispatchID != "" {
		ticket, err = dispatch.EnsureCurrent(request.Root, request.DispatchID, request.TaskID, request.PlanEpoch)
		if err != nil {
			return Result{}, err
		}
		ticketFound = true
	}
	task, taskErr := adapter.LoadTask(request.Root, request.TaskID)
	taskFound := taskErr == nil
	payload, err := a2a.NewPayload(map[string]any{
		"status":                 request.Status,
		"summary":                request.Summary,
		"verificationResultPath": request.VerificationResultPath,
	})
	if err != nil {
		return Result{}, err
	}
	verificationKey := fmt.Sprintf("verification:%s:%d", request.TaskID, request.Attempt)
	verificationResult, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
		Kind:           "verification.completed",
		IdempotencyKey: verificationKey,
		ProjectID:      task.ProjectID,
		ProjectSpaceID: task.ProjectSpaceID,
		TraceID:        request.RequestID,
		CausationID:    request.CausationID,
		From:           "orchestrator-node",
		To:             "worker-supervisor-node",
		RequestID:      request.RequestID,
		TaskID:         request.TaskID,
		PlanEpoch:      request.PlanEpoch,
		Attempt:        request.Attempt,
		ReasonCodes:    request.ReasonCodes,
		Payload:        payload,
	})
	if err != nil {
		return Result{}, err
	}
	if passedVerificationStatus(request.Status) {
		if err := recordCompletedExecutionSlice(request.Root, request.TaskID, request.DispatchID); err != nil {
			return Result{}, err
		}
	}
	result := Result{VerificationEvent: verificationResult.Event.Kind}
	gate, err := updateCompletionState(paths, request, task, taskFound, ticket, ticketFound)
	if err != nil {
		return Result{}, err
	}
	switch request.Status {
	case "passed", "succeeded", "verified", "already_satisfied", "noop_verified":
		if request.DispatchID != "" {
			if _, err := dispatch.UpdateStatus(request.Root, request.DispatchID, "verified", "kh-orchestrator"); err != nil {
				return Result{}, err
			}
		}
		if !gate.Satisfied {
			if executionTasksRemainingOnly(gate) {
				payload, err := a2a.NewPayload(map[string]any{
					"sourceTaskId": request.TaskID,
					"followUpKind": "replan",
					"summary":      "current execution slice verified; additional execution slices remain",
				})
				if err != nil {
					return Result{}, err
				}
				if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
					Kind:           "replan.emitted",
					IdempotencyKey: fmt.Sprintf("replan:%s:%d", request.TaskID, request.Attempt),
					ProjectID:      task.ProjectID,
					ProjectSpaceID: task.ProjectSpaceID,
					TraceID:        request.RequestID,
					CausationID:    verificationResult.Event.MessageID,
					From:           "orchestrator-node",
					To:             "worker-supervisor-node",
					RequestID:      request.RequestID,
					TaskID:         request.TaskID,
					PlanEpoch:      request.PlanEpoch,
					Attempt:        request.Attempt,
					ReasonCodes:    request.ReasonCodes,
					Payload:        payload,
				}); err != nil {
					return Result{}, err
				}
				result.FollowUpEvent = "replan.emitted"
				return result, nil
			}
			return Result{}, completionGateError(request, gate)
		}
		completionMode := "verified"
		if request.Status == "already_satisfied" || request.Status == "noop_verified" {
			completionMode = "noop_verified"
		}
		completePayload, err := a2a.NewPayload(map[string]any{
			"status":         "completed",
			"summary":        request.Summary,
			"completionMode": completionMode,
		})
		if err != nil {
			return Result{}, err
		}
		if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
			Kind:           "task.completed",
			IdempotencyKey: fmt.Sprintf("completed:%s:%d", request.TaskID, request.Attempt),
			ProjectID:      task.ProjectID,
			ProjectSpaceID: task.ProjectSpaceID,
			TraceID:        request.RequestID,
			CausationID:    verificationResult.Event.MessageID,
			From:           "orchestrator-node",
			To:             "worker-supervisor-node",
			RequestID:      request.RequestID,
			TaskID:         request.TaskID,
			PlanEpoch:      request.PlanEpoch,
			Attempt:        request.Attempt,
			ReasonCodes:    request.ReasonCodes,
			Payload:        completePayload,
		}); err != nil {
			return Result{}, err
		}
		result.FollowUpEvent = "task.completed"
		return result, nil
	case "blocked":
		blockPayload, err := a2a.NewPayload(map[string]any{"status": "blocked", "summary": request.Summary})
		if err != nil {
			return Result{}, err
		}
		if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
			Kind:           "task.blocked",
			IdempotencyKey: fmt.Sprintf("blocked:%s:%d", request.TaskID, request.Attempt),
			ProjectID:      task.ProjectID,
			ProjectSpaceID: task.ProjectSpaceID,
			TraceID:        request.RequestID,
			CausationID:    verificationResult.Event.MessageID,
			From:           "orchestrator-node",
			To:             "worker-supervisor-node",
			RequestID:      request.RequestID,
			TaskID:         request.TaskID,
			PlanEpoch:      request.PlanEpoch,
			Attempt:        request.Attempt,
			ReasonCodes:    request.ReasonCodes,
			Payload:        blockPayload,
		}); err != nil {
			return Result{}, err
		}
		result.FollowUpEvent = "task.blocked"
		return result, nil
	}

	if request.FollowUp == "rca" {
		allocation := rca.Allocate(request.Summary, request.ReasonCodes)
		payload, err := a2a.NewPayload(map[string]any{
			"sourceTaskId": request.TaskID,
			"taxonomy":     allocation.Taxonomy,
			"ownerRole":    allocation.OwnerRole,
			"summary":      allocation.Summary,
		})
		if err != nil {
			return Result{}, err
		}
		if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
			Kind:           "rca.allocated",
			IdempotencyKey: fmt.Sprintf("rca:%s:%d", request.TaskID, request.Attempt),
			ProjectID:      task.ProjectID,
			ProjectSpaceID: task.ProjectSpaceID,
			TraceID:        request.RequestID,
			CausationID:    verificationResult.Event.MessageID,
			From:           "orchestrator-node",
			To:             "worker-supervisor-node",
			RequestID:      request.RequestID,
			TaskID:         request.TaskID,
			PlanEpoch:      request.PlanEpoch,
			Attempt:        request.Attempt,
			ReasonCodes:    request.ReasonCodes,
			Payload:        payload,
		}); err != nil {
			return Result{}, err
		}
		result.FollowUpEvent = "rca.allocated"
		return result, nil
	}

	payload, err = a2a.NewPayload(map[string]any{
		"sourceTaskId": request.TaskID,
		"followUpKind": "replan",
		"summary":      request.Summary,
	})
	if err != nil {
		return Result{}, err
	}
	if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
		Kind:           "replan.emitted",
		IdempotencyKey: fmt.Sprintf("replan:%s:%d", request.TaskID, request.Attempt),
		ProjectID:      task.ProjectID,
		ProjectSpaceID: task.ProjectSpaceID,
		TraceID:        request.RequestID,
		CausationID:    verificationResult.Event.MessageID,
		From:           "orchestrator-node",
		To:             "worker-supervisor-node",
		RequestID:      request.RequestID,
		TaskID:         request.TaskID,
		PlanEpoch:      request.PlanEpoch,
		Attempt:        request.Attempt,
		ReasonCodes:    request.ReasonCodes,
		Payload:        payload,
	}); err != nil {
		return Result{}, err
	}
	result.FollowUpEvent = "replan.emitted"
	return result, nil
}

func executionTasksRemainingOnly(gate CompletionGate) bool {
	if gate.Satisfied {
		return false
	}
	failed := 0
	for name, check := range gate.Checks {
		if !check.OK {
			failed++
			if name != "executionTasks" {
				return false
			}
		}
	}
	return failed == 1
}
