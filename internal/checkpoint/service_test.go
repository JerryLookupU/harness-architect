package checkpoint

import (
	"testing"

	"klein-harness/internal/dispatch"
	"klein-harness/internal/lease"
)

func TestIngestCheckpointAndOutcome(t *testing.T) {
	root := t.TempDir()
	ticket, _, err := dispatch.Issue(dispatch.IssueRequest{
		Root:           root,
		TaskID:         "T-1",
		ThreadKey:      "thread-1",
		PlanEpoch:      1,
		Attempt:        1,
		IdempotencyKey: "dispatch:T-1:epoch_1:attempt_1",
		CausationID:    "route-1",
		WorkerClass:    "codex-go",
		Cwd:            root,
		Command:        "printf ok",
		PromptRef:      "prompts/worker-burst.md",
	})
	if err != nil {
		t.Fatalf("issue dispatch: %v", err)
	}
	leaseRecord, err := lease.Acquire(lease.AcquireRequest{
		Root:        root,
		TaskID:      ticket.TaskID,
		DispatchID:  ticket.DispatchID,
		WorkerID:    "worker-1",
		LeaseID:     "lease-1",
		CausationID: "claim-1",
	})
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	checkpointRecord, err := IngestCheckpoint(IngestCheckpointRequest{
		Root:          root,
		TaskID:        ticket.TaskID,
		DispatchID:    ticket.DispatchID,
		PlanEpoch:     ticket.PlanEpoch,
		Attempt:       1,
		CausationID:   "claim-1",
		ThreadKey:     ticket.ThreadKey,
		LeaseID:       leaseRecord.LeaseID,
		CheckpointRef: ".harness/checkpoints/T-1/attempt_1.json",
		Status:        "checkpointed",
		Summary:       "safe boundary",
	})
	if err != nil {
		t.Fatalf("ingest checkpoint: %v", err)
	}
	if checkpointRecord.Status != "checkpointed" {
		t.Fatalf("unexpected checkpoint status %s", checkpointRecord.Status)
	}
	outcome, err := IngestOutcome(IngestOutcomeRequest{
		Root:          root,
		TaskID:        ticket.TaskID,
		DispatchID:    ticket.DispatchID,
		PlanEpoch:     ticket.PlanEpoch,
		Attempt:       1,
		CausationID:   "burst-1",
		WorkerID:      "worker-1",
		LeaseID:       leaseRecord.LeaseID,
		ThreadKey:     ticket.ThreadKey,
		Status:        "needs_replan",
		Summary:       "verification drift",
		CheckpointRef: checkpointRecord.CheckpointRef,
	})
	if err != nil {
		t.Fatalf("ingest outcome: %v", err)
	}
	if outcome.Status != "needs_replan" {
		t.Fatalf("unexpected outcome status %s", outcome.Status)
	}
}
