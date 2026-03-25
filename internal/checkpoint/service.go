package checkpoint

import (
	"fmt"

	"klein-harness/internal/a2a"
	"klein-harness/internal/adapter"
	"klein-harness/internal/dispatch"
	"klein-harness/internal/lease"
	"klein-harness/internal/state"
)

type DiffStats struct {
	FilesChanged int `json:"filesChanged"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
}

type CheckpointRecord struct {
	DispatchID    string   `json:"dispatchId"`
	TaskID        string   `json:"taskId"`
	ThreadKey     string   `json:"threadKey,omitempty"`
	PlanEpoch     int      `json:"planEpoch"`
	Attempt       int      `json:"attempt"`
	LeaseID       string   `json:"leaseId,omitempty"`
	CheckpointRef string   `json:"checkpointRef"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	ReasonCodes   []string `json:"reasonCodes,omitempty"`
	UpdatedAt     string   `json:"updatedAt"`
}

type OutcomeRecord struct {
	DispatchID        string    `json:"dispatchId"`
	TaskID            string    `json:"taskId"`
	ThreadKey         string    `json:"threadKey,omitempty"`
	PlanEpoch         int       `json:"planEpoch"`
	Attempt           int       `json:"attempt"`
	WorkerID          string    `json:"workerId,omitempty"`
	LeaseID           string    `json:"leaseId,omitempty"`
	Status            string    `json:"status"`
	Summary           string    `json:"summary"`
	CheckpointRef     string    `json:"checkpointRef,omitempty"`
	DiffStats         DiffStats `json:"diffStats"`
	Artifacts         []string  `json:"artifacts,omitempty"`
	NextSuggestedKind string    `json:"nextSuggestedKind,omitempty"`
	UpdatedAt         string    `json:"updatedAt"`
}

type TaskState struct {
	TaskID           string           `json:"taskId"`
	LatestCheckpoint CheckpointRecord `json:"latestCheckpoint"`
	LatestOutcome    OutcomeRecord    `json:"latestOutcome"`
}

type Summary struct {
	state.Metadata
	Tasks      map[string]TaskState `json:"tasks"`
	ByDispatch map[string]string    `json:"byDispatch,omitempty"`
}

type IngestCheckpointRequest struct {
	Root          string
	RequestID     string
	TaskID        string
	DispatchID    string
	PlanEpoch     int
	Attempt       int
	CausationID   string
	ReasonCodes   []string
	ThreadKey     string
	LeaseID       string
	CheckpointRef string
	Status        string
	Summary       string
}

type IngestOutcomeRequest struct {
	Root              string
	RequestID         string
	TaskID            string
	DispatchID        string
	PlanEpoch         int
	Attempt           int
	CausationID       string
	WorkerID          string
	LeaseID           string
	ReasonCodes       []string
	ThreadKey         string
	Status            string
	Summary           string
	CheckpointRef     string
	DiffStats         DiffStats
	Artifacts         []string
	NextSuggestedKind string
}

func IngestCheckpoint(request IngestCheckpointRequest) (CheckpointRecord, error) {
	paths, err := adapter.Resolve(request.Root)
	if err != nil {
		return CheckpointRecord{}, err
	}
	summary, err := loadSummary(paths.CheckpointSummaryPath)
	if err != nil {
		return CheckpointRecord{}, err
	}
	if _, err := dispatch.EnsureCurrent(request.Root, request.DispatchID, request.TaskID, request.PlanEpoch); err != nil {
		return CheckpointRecord{}, err
	}
	if request.LeaseID != "" {
		if _, err := lease.ValidateCurrent(request.Root, request.LeaseID, request.TaskID, request.DispatchID); err != nil {
			return CheckpointRecord{}, err
		}
	}
	record := CheckpointRecord{
		DispatchID:    request.DispatchID,
		TaskID:        request.TaskID,
		ThreadKey:     request.ThreadKey,
		PlanEpoch:     request.PlanEpoch,
		Attempt:       request.Attempt,
		LeaseID:       request.LeaseID,
		CheckpointRef: request.CheckpointRef,
		Status:        request.Status,
		Summary:       request.Summary,
		ReasonCodes:   request.ReasonCodes,
		UpdatedAt:     state.NowUTC(),
	}
	taskState := summary.Tasks[request.TaskID]
	taskState.TaskID = request.TaskID
	taskState.LatestCheckpoint = record
	summary.Tasks[request.TaskID] = taskState
	summary.ByDispatch[request.DispatchID] = request.TaskID
	if _, err := state.WriteSnapshot(paths.CheckpointSummaryPath, &summary, "kh-worker-supervisor", summary.Revision); err != nil {
		return CheckpointRecord{}, err
	}
	payload, err := a2a.NewPayload(record)
	if err != nil {
		return CheckpointRecord{}, err
	}
	if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
		Kind:           "worker.checkpoint",
		IdempotencyKey: fmt.Sprintf("checkpoint:%s:%d", request.TaskID, request.Attempt),
		TraceID:        request.RequestID,
		CausationID:    request.CausationID,
		From:           "worker-supervisor-node",
		To:             "orchestrator-node",
		RequestID:      request.RequestID,
		TaskID:         request.TaskID,
		PlanEpoch:      request.PlanEpoch,
		Attempt:        request.Attempt,
		ReasonCodes:    request.ReasonCodes,
		Payload:        payload,
	}); err != nil {
		return CheckpointRecord{}, err
	}
	return record, nil
}

func IngestOutcome(request IngestOutcomeRequest) (OutcomeRecord, error) {
	paths, err := adapter.Resolve(request.Root)
	if err != nil {
		return OutcomeRecord{}, err
	}
	summary, err := loadSummary(paths.CheckpointSummaryPath)
	if err != nil {
		return OutcomeRecord{}, err
	}
	if _, err := dispatch.EnsureCurrent(request.Root, request.DispatchID, request.TaskID, request.PlanEpoch); err != nil {
		return OutcomeRecord{}, err
	}
	if _, err := lease.ValidateCurrent(request.Root, request.LeaseID, request.TaskID, request.DispatchID); err != nil {
		return OutcomeRecord{}, err
	}
	record := OutcomeRecord{
		DispatchID:        request.DispatchID,
		TaskID:            request.TaskID,
		ThreadKey:         request.ThreadKey,
		PlanEpoch:         request.PlanEpoch,
		Attempt:           request.Attempt,
		WorkerID:          request.WorkerID,
		LeaseID:           request.LeaseID,
		Status:            request.Status,
		Summary:           request.Summary,
		CheckpointRef:     request.CheckpointRef,
		DiffStats:         request.DiffStats,
		Artifacts:         request.Artifacts,
		NextSuggestedKind: request.NextSuggestedKind,
		UpdatedAt:         state.NowUTC(),
	}
	taskState := summary.Tasks[request.TaskID]
	taskState.TaskID = request.TaskID
	taskState.LatestOutcome = record
	summary.Tasks[request.TaskID] = taskState
	summary.ByDispatch[request.DispatchID] = request.TaskID
	if _, err := state.WriteSnapshot(paths.CheckpointSummaryPath, &summary, "kh-worker-supervisor", summary.Revision); err != nil {
		return OutcomeRecord{}, err
	}
	payload, err := a2a.NewPayload(record)
	if err != nil {
		return OutcomeRecord{}, err
	}
	if _, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
		Kind:           "worker.outcome",
		IdempotencyKey: fmt.Sprintf("outcome:%s:%d", request.TaskID, request.Attempt),
		TraceID:        request.RequestID,
		CausationID:    request.CausationID,
		From:           "worker-supervisor-node",
		To:             "orchestrator-node",
		RequestID:      request.RequestID,
		TaskID:         request.TaskID,
		PlanEpoch:      request.PlanEpoch,
		Attempt:        request.Attempt,
		WorkerID:       request.WorkerID,
		LeaseID:        request.LeaseID,
		ReasonCodes:    request.ReasonCodes,
		Payload:        payload,
	}); err != nil {
		return OutcomeRecord{}, err
	}
	return record, nil
}

func loadSummary(path string) (Summary, error) {
	summary := Summary{
		Tasks:      map[string]TaskState{},
		ByDispatch: map[string]string{},
	}
	if _, err := state.LoadJSONIfExists(path, &summary); err != nil {
		return Summary{}, err
	}
	if summary.Tasks == nil {
		summary.Tasks = map[string]TaskState{}
	}
	if summary.ByDispatch == nil {
		summary.ByDispatch = map[string]string{}
	}
	return summary, nil
}
