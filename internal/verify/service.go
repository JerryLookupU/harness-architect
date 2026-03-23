package verify

import (
	"fmt"

	"klein-harness/internal/a2a"
	"klein-harness/internal/adapter"
	"klein-harness/internal/dispatch"
	"klein-harness/internal/rca"
)

type Request struct {
	Root                 string
	RequestID            string
	TaskID               string
	DispatchID           string
	PlanEpoch            int
	Attempt              int
	CausationID          string
	ReasonCodes          []string
	Status               string
	Summary              string
	VerificationResultPath string
	FollowUp             string
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
	result := Result{VerificationEvent: verificationResult.Event.Kind}
	switch request.Status {
	case "passed", "succeeded", "verified":
		if _, err := dispatch.UpdateStatus(request.Root, request.DispatchID, "verified", "kh-orchestrator"); err != nil {
			return Result{}, err
		}
		completePayload, err := a2a.NewPayload(map[string]any{"status": "completed", "summary": request.Summary})
		if err != nil {
			return Result{}, err
		}
		if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
			Kind:           "task.completed",
			IdempotencyKey: fmt.Sprintf("completed:%s:%d", request.TaskID, request.Attempt),
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
