package checkpoint

import (
	"fmt"

	"klein-harness/internal/a2a"
	"klein-harness/internal/adapter"
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
	Attempt       int      `json:"attempt"`
	CheckpointRef string   `json:"checkpointRef"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	ReasonCodes   []string `json:"reasonCodes,omitempty"`
	UpdatedAt     string   `json:"updatedAt"`
}

type OutcomeRecord struct {
	DispatchID        string    `json:"dispatchId"`
	TaskID            string    `json:"taskId"`
	Attempt           int       `json:"attempt"`
	Status            string    `json:"status"`
	Summary           string    `json:"summary"`
	CheckpointRef     string    `json:"checkpointRef,omitempty"`
	DiffStats         DiffStats `json:"diffStats"`
	Artifacts         []string  `json:"artifacts,omitempty"`
	NextSuggestedKind string    `json:"nextSuggestedKind,omitempty"`
	UpdatedAt         string    `json:"updatedAt"`
}

type TaskState struct {
	TaskID            string           `json:"taskId"`
	LatestCheckpoint  CheckpointRecord `json:"latestCheckpoint"`
	LatestOutcome     OutcomeRecord    `json:"latestOutcome"`
}

type Summary struct {
	state.Metadata
	Tasks map[string]TaskState `json:"tasks"`
}

type IngestCheckpointRequest struct {
	Root         string
	RequestID    string
	TaskID       string
	DispatchID   string
	PlanEpoch    int
	Attempt      int
	CausationID  string
	ReasonCodes  []string
	CheckpointRef string
	Status       string
	Summary      string
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
	record := CheckpointRecord{
		DispatchID:    request.DispatchID,
		TaskID:        request.TaskID,
		Attempt:       request.Attempt,
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
	record := OutcomeRecord{
		DispatchID:        request.DispatchID,
		TaskID:            request.TaskID,
		Attempt:           request.Attempt,
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
		Tasks: map[string]TaskState{},
	}
	if _, err := state.LoadJSONIfExists(path, &summary); err != nil {
		return Summary{}, err
	}
	if summary.Tasks == nil {
		summary.Tasks = map[string]TaskState{}
	}
	return summary, nil
}
