package checkpoint

import "testing"

func TestIngestCheckpointAndOutcome(t *testing.T) {
	root := t.TempDir()
	checkpointRecord, err := IngestCheckpoint(IngestCheckpointRequest{
		Root:          root,
		TaskID:        "T-1",
		DispatchID:    "dispatch-1",
		Attempt:       1,
		CausationID:   "claim-1",
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
		TaskID:        "T-1",
		DispatchID:    "dispatch-1",
		Attempt:       1,
		CausationID:   "burst-1",
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
