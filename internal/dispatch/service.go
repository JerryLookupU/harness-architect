package dispatch

import (
	"errors"
	"fmt"
	"path/filepath"

	"klein-harness/internal/a2a"
	"klein-harness/internal/adapter"
	"klein-harness/internal/state"
)

var ErrDispatchClaimed = errors.New("dispatch already claimed by another worker")
var ErrDispatchNotFound = errors.New("dispatch not found")
var ErrDispatchStale = errors.New("dispatch is stale for current task execution")

type Budget struct {
	MaxTurns     int `json:"maxTurns"`
	MaxMinutes   int `json:"maxMinutes"`
	MaxToolCalls int `json:"maxToolCalls"`
}

type Ticket struct {
	DispatchID             string   `json:"dispatchId"`
	IdempotencyKey         string   `json:"idempotencyKey,omitempty"`
	RequestID              string   `json:"requestId,omitempty"`
	TaskID                 string   `json:"taskId"`
	ProjectID              string   `json:"projectId,omitempty"`
	ProjectSpaceID         string   `json:"projectSpaceId,omitempty"`
	ThreadKey              string   `json:"threadKey,omitempty"`
	PlanEpoch              int      `json:"planEpoch"`
	Attempt                int      `json:"attempt"`
	Status                 string   `json:"status"`
	WorkerClass            string   `json:"workerClass"`
	Cwd                    string   `json:"cwd"`
	Command                string   `json:"command"`
	PromptRef              string   `json:"promptRef"`
	Budget                 Budget   `json:"budget"`
	LeaseTTLSec            int      `json:"leaseTtlSec"`
	CausationID            string   `json:"causationId"`
	ReasonCodes            []string `json:"reasonCodes,omitempty"`
	RequiredSummaryVersion string   `json:"requiredSummaryVersion,omitempty"`
	ResumeSessionID        string   `json:"resumeSessionId,omitempty"`
	WorktreePath           string   `json:"worktreePath,omitempty"`
	OwnedPaths             []string `json:"ownedPaths,omitempty"`
	ClaimedBy              string   `json:"claimedBy,omitempty"`
	LeaseID                string   `json:"leaseId,omitempty"`
	CreatedAt              string   `json:"createdAt"`
	UpdatedAt              string   `json:"updatedAt"`
}

type Summary struct {
	state.Metadata
	Tickets          map[string]Ticket   `json:"tickets"`
	IdempotencyIndex map[string]string   `json:"idempotencyIndex"`
	TaskIndex        map[string][]string `json:"taskIndex"`
	LatestByTask     map[string]string   `json:"latestByTask"`
	ThreadEpochIndex map[string]string   `json:"threadEpochIndex,omitempty"`
}

type IssueRequest struct {
	Root                   string
	RequestID              string
	TaskID                 string
	ThreadKey              string
	PlanEpoch              int
	Attempt                int
	IdempotencyKey         string
	CausationID            string
	ReasonCodes            []string
	WorkerClass            string
	Cwd                    string
	Command                string
	PromptRef              string
	Budget                 Budget
	LeaseTTLSec            int
	RequiredSummaryVersion string
	ResumeSessionID        string
	WorktreePath           string
	OwnedPaths             []string
}

type ClaimRequest struct {
	Root        string
	DispatchID  string
	TaskID      string
	WorkerID    string
	LeaseID     string
	CausationID string
	ReasonCodes []string
}

func Issue(request IssueRequest) (Ticket, bool, error) {
	paths, err := adapter.Resolve(request.Root)
	if err != nil {
		return Ticket{}, false, err
	}
	projectTask, _ := adapter.LoadTask(request.Root, request.TaskID)
	summary, err := loadSummary(paths.DispatchSummaryPath)
	if err != nil {
		return Ticket{}, false, err
	}
	if request.Attempt <= 0 {
		request.Attempt = 1
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = fmt.Sprintf("dispatch:%s:epoch_%d:attempt_%d", request.TaskID, request.PlanEpoch, request.Attempt)
	}
	if existingID := summary.IdempotencyIndex[request.IdempotencyKey]; existingID != "" {
		return summary.Tickets[existingID], true, nil
	}
	dispatchID := fmt.Sprintf("dispatch_%s_%d_%d", request.TaskID, request.PlanEpoch, request.Attempt)
	now := state.NowUTC()
	ticket := Ticket{
		DispatchID:             dispatchID,
		IdempotencyKey:         request.IdempotencyKey,
		RequestID:              request.RequestID,
		TaskID:                 request.TaskID,
		ProjectID:              projectTask.ProjectID,
		ProjectSpaceID:         projectTask.ProjectSpaceID,
		ThreadKey:              request.ThreadKey,
		PlanEpoch:              request.PlanEpoch,
		Attempt:                request.Attempt,
		Status:                 "issued",
		WorkerClass:            request.WorkerClass,
		Cwd:                    request.Cwd,
		Command:                request.Command,
		PromptRef:              request.PromptRef,
		Budget:                 request.Budget,
		LeaseTTLSec:            request.LeaseTTLSec,
		CausationID:            request.CausationID,
		ReasonCodes:            request.ReasonCodes,
		RequiredSummaryVersion: request.RequiredSummaryVersion,
		ResumeSessionID:        request.ResumeSessionID,
		WorktreePath:           request.WorktreePath,
		OwnedPaths:             request.OwnedPaths,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	summary.Tickets[dispatchID] = ticket
	summary.IdempotencyIndex[request.IdempotencyKey] = dispatchID
	summary.TaskIndex[request.TaskID] = append(summary.TaskIndex[request.TaskID], dispatchID)
	summary.LatestByTask[request.TaskID] = dispatchID
	if request.ThreadKey != "" && request.PlanEpoch > 0 {
		summary.ThreadEpochIndex[threadEpochKey(request.ThreadKey, request.PlanEpoch)] = dispatchID
	}
	if _, err := state.WriteSnapshot(paths.DispatchSummaryPath, &summary, "kh-orchestrator", summary.Revision); err != nil {
		return Ticket{}, false, err
	}
	payload, err := a2a.NewPayload(map[string]any{
		"workerClass": request.WorkerClass,
		"cwd":         request.Cwd,
		"command":     request.Command,
		"promptRef":   request.PromptRef,
		"budget":      request.Budget,
		"leaseTtlSec": request.LeaseTTLSec,
	})
	if err != nil {
		return Ticket{}, false, err
	}
	if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
		Kind:           "dispatch.issued",
		IdempotencyKey: request.IdempotencyKey,
		ProjectID:      ticket.ProjectID,
		ProjectSpaceID: ticket.ProjectSpaceID,
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
	}); err != nil {
		return Ticket{}, false, err
	}
	return ticket, false, nil
}

func Claim(request ClaimRequest) (Ticket, error) {
	paths, err := adapter.Resolve(request.Root)
	if err != nil {
		return Ticket{}, err
	}
	summary, err := loadSummary(paths.DispatchSummaryPath)
	if err != nil {
		return Ticket{}, err
	}
	ticket, ok := findTicket(summary, request.DispatchID, request.TaskID)
	if !ok {
		return Ticket{}, ErrDispatchNotFound
	}
	if latest := summary.LatestByTask[ticket.TaskID]; latest != "" && latest != ticket.DispatchID {
		return Ticket{}, fmt.Errorf("%w: latest dispatch for %s is %s", ErrDispatchStale, ticket.TaskID, latest)
	}
	if ticket.ThreadKey != "" && ticket.PlanEpoch > 0 {
		if latest := summary.ThreadEpochIndex[threadEpochKey(ticket.ThreadKey, ticket.PlanEpoch)]; latest != "" && latest != ticket.DispatchID {
			return Ticket{}, fmt.Errorf("%w: latest dispatch for %s is %s", ErrDispatchStale, threadEpochKey(ticket.ThreadKey, ticket.PlanEpoch), latest)
		}
	}
	if ticket.Status == "claimed" && ticket.ClaimedBy != request.WorkerID {
		return Ticket{}, ErrDispatchClaimed
	}
	if ticket.Status == "claimed" && ticket.ClaimedBy == request.WorkerID && ticket.LeaseID == request.LeaseID {
		return ticket, nil
	}
	ticket.Status = "claimed"
	ticket.ClaimedBy = request.WorkerID
	ticket.LeaseID = request.LeaseID
	ticket.UpdatedAt = state.NowUTC()
	summary.Tickets[ticket.DispatchID] = ticket
	if _, err := state.WriteSnapshot(paths.DispatchSummaryPath, &summary, "kh-worker-supervisor", summary.Revision); err != nil {
		return Ticket{}, err
	}
	payload, err := a2a.NewPayload(map[string]any{
		"dispatchId": ticket.DispatchID,
		"workerId":   request.WorkerID,
		"leaseId":    request.LeaseID,
		"claimedAt":  ticket.UpdatedAt,
	})
	if err != nil {
		return Ticket{}, err
	}
	if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
		Kind:           "worker.claimed",
		IdempotencyKey: fmt.Sprintf("claim:%s:%s", ticket.DispatchID, request.LeaseID),
		ProjectID:      ticket.ProjectID,
		ProjectSpaceID: ticket.ProjectSpaceID,
		TraceID:        ticket.RequestID,
		CausationID:    request.CausationID,
		From:           "worker-supervisor-node",
		To:             "orchestrator-node",
		RequestID:      ticket.RequestID,
		TaskID:         ticket.TaskID,
		PlanEpoch:      ticket.PlanEpoch,
		Attempt:        ticket.Attempt,
		WorkerID:       request.WorkerID,
		LeaseID:        request.LeaseID,
		ReasonCodes:    request.ReasonCodes,
		Payload:        payload,
	}); err != nil {
		return Ticket{}, err
	}
	return ticket, nil
}

func UpdateStatus(root, dispatchID, status, generator string) (Ticket, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return Ticket{}, err
	}
	summary, err := loadSummary(paths.DispatchSummaryPath)
	if err != nil {
		return Ticket{}, err
	}
	ticket, ok := summary.Tickets[dispatchID]
	if !ok {
		return Ticket{}, ErrDispatchNotFound
	}
	ticket.Status = status
	ticket.UpdatedAt = state.NowUTC()
	summary.Tickets[dispatchID] = ticket
	if _, err := state.WriteSnapshot(paths.DispatchSummaryPath, &summary, generator, summary.Revision); err != nil {
		return Ticket{}, err
	}
	return ticket, nil
}

func Get(root, dispatchID string) (Ticket, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return Ticket{}, err
	}
	summary, err := loadSummary(paths.DispatchSummaryPath)
	if err != nil {
		return Ticket{}, err
	}
	ticket, ok := summary.Tickets[dispatchID]
	if !ok {
		return Ticket{}, ErrDispatchNotFound
	}
	return ticket, nil
}

func FindClaimableForTask(root, taskID string) (Ticket, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return Ticket{}, err
	}
	summary, err := loadSummary(paths.DispatchSummaryPath)
	if err != nil {
		return Ticket{}, err
	}
	ids := summary.TaskIndex[taskID]
	for index := len(ids) - 1; index >= 0; index-- {
		ticket := summary.Tickets[ids[index]]
		if ticket.Status == "issued" || ticket.Status == "claimed" {
			return ticket, nil
		}
	}
	return Ticket{}, ErrDispatchNotFound
}

func EnsureCurrent(root, dispatchID, taskID string, planEpoch int) (Ticket, error) {
	ticket, err := Get(root, dispatchID)
	if err != nil {
		return Ticket{}, err
	}
	if taskID != "" && ticket.TaskID != taskID {
		return Ticket{}, fmt.Errorf("%w: task mismatch %s != %s", ErrDispatchStale, ticket.TaskID, taskID)
	}
	if planEpoch > 0 && ticket.PlanEpoch != planEpoch {
		return Ticket{}, fmt.Errorf("%w: plan epoch mismatch %d != %d", ErrDispatchStale, ticket.PlanEpoch, planEpoch)
	}
	paths, err := adapter.Resolve(root)
	if err != nil {
		return Ticket{}, err
	}
	summary, err := loadSummary(paths.DispatchSummaryPath)
	if err != nil {
		return Ticket{}, err
	}
	if latest := summary.LatestByTask[ticket.TaskID]; latest != "" && latest != dispatchID {
		return Ticket{}, fmt.Errorf("%w: latest dispatch for %s is %s", ErrDispatchStale, ticket.TaskID, latest)
	}
	if ticket.ThreadKey != "" && ticket.PlanEpoch > 0 {
		if latest := summary.ThreadEpochIndex[threadEpochKey(ticket.ThreadKey, ticket.PlanEpoch)]; latest != "" && latest != dispatchID {
			return Ticket{}, fmt.Errorf("%w: latest dispatch for %s is %s", ErrDispatchStale, threadEpochKey(ticket.ThreadKey, ticket.PlanEpoch), latest)
		}
	}
	return ticket, nil
}

func loadSummary(path string) (Summary, error) {
	summary := Summary{
		Tickets:          map[string]Ticket{},
		IdempotencyIndex: map[string]string{},
		TaskIndex:        map[string][]string{},
		LatestByTask:     map[string]string{},
		ThreadEpochIndex: map[string]string{},
	}
	if _, err := state.LoadJSONIfExists(path, &summary); err != nil {
		return Summary{}, err
	}
	if summary.Tickets == nil {
		summary.Tickets = map[string]Ticket{}
	}
	if summary.IdempotencyIndex == nil {
		summary.IdempotencyIndex = map[string]string{}
	}
	if summary.TaskIndex == nil {
		summary.TaskIndex = map[string][]string{}
	}
	if summary.LatestByTask == nil {
		summary.LatestByTask = map[string]string{}
	}
	if summary.ThreadEpochIndex == nil {
		summary.ThreadEpochIndex = map[string]string{}
	}
	if len(summary.LatestByTask) == 0 {
		for taskID, ids := range summary.TaskIndex {
			if len(ids) == 0 {
				continue
			}
			summary.LatestByTask[taskID] = ids[len(ids)-1]
		}
	}
	if len(summary.ThreadEpochIndex) == 0 {
		for _, ticket := range summary.Tickets {
			if ticket.ThreadKey == "" || ticket.PlanEpoch <= 0 {
				continue
			}
			summary.ThreadEpochIndex[threadEpochKey(ticket.ThreadKey, ticket.PlanEpoch)] = ticket.DispatchID
		}
	}
	return summary, nil
}

func findTicket(summary Summary, dispatchID, taskID string) (Ticket, bool) {
	if dispatchID != "" {
		ticket, ok := summary.Tickets[dispatchID]
		return ticket, ok
	}
	if taskID == "" {
		return Ticket{}, false
	}
	ids := summary.TaskIndex[taskID]
	for index := len(ids) - 1; index >= 0; index-- {
		ticket := summary.Tickets[ids[index]]
		if ticket.Status == "issued" || ticket.Status == "claimed" {
			return ticket, true
		}
	}
	return Ticket{}, false
}

func DefaultCheckpointPath(root, taskID string, attempt int) string {
	return filepath.Join(root, ".harness", "checkpoints", taskID, fmt.Sprintf("attempt_%d.json", attempt))
}

func threadEpochKey(threadKey string, planEpoch int) string {
	return fmt.Sprintf("%s:%d", threadKey, planEpoch)
}
