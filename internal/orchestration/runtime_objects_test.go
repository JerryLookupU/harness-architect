package orchestration

import (
	"errors"
	"path/filepath"
	"testing"

	"klein-harness/internal/state"
)

func TestWriteAcceptedPacketCASIncrementsRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accepted-packet.json")
	packet := AcceptedPacket{
		SchemaVersion:     "kh.accepted-packet.v1",
		Generator:         "test",
		GeneratedAt:       "2026-03-27T00:00:00Z",
		TaskID:            "T-1",
		ThreadKey:         "thread-1",
		PlanEpoch:         1,
		PacketID:          "packet_T-1_1",
		Objective:         "first",
		SelectedPlan:      "plan",
		ExecutionTasks:    []ExecutionTask{{ID: "T-1.slice.1", Title: "slice", Summary: "slice"}},
		VerificationPlan:  map[string]any{},
		DecisionRationale: "test",
		AcceptedAt:        "2026-03-27T00:00:00Z",
		AcceptedBy:        "test",
	}
	if err := WriteAcceptedPacketCAS(path, packet, 0); err != nil {
		t.Fatalf("first packet write: %v", err)
	}
	written, err := LoadAcceptedPacket(path)
	if err != nil {
		t.Fatalf("load first packet: %v", err)
	}
	if written.Revision != 1 {
		t.Fatalf("expected revision 1 after first write, got %+v", written)
	}

	packet.Objective = "second"
	if err := WriteAcceptedPacketCAS(path, packet, 1); err != nil {
		t.Fatalf("second packet write: %v", err)
	}
	written, err = LoadAcceptedPacket(path)
	if err != nil {
		t.Fatalf("load second packet: %v", err)
	}
	if written.Revision != 2 || written.Objective != "second" {
		t.Fatalf("expected revision 2 packet after second write, got %+v", written)
	}

	if err := WriteAcceptedPacketCAS(path, packet, 1); !errors.Is(err, state.ErrCASConflict) {
		t.Fatalf("expected stale packet write to fail with CAS conflict, got %v", err)
	}
}

func TestWriteTaskContractCASIncrementsRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task-contract.json")
	contract := TaskContract{
		SchemaVersion:    "kh.task-contract.v1",
		Generator:        "test",
		GeneratedAt:      "2026-03-27T00:00:00Z",
		ContractID:       "contract_T-1_1_1",
		TaskID:           "T-1",
		DispatchID:       "dispatch-1",
		ThreadKey:        "thread-1",
		PlanEpoch:        1,
		ExecutionSliceID: "T-1.slice.1",
		Objective:        "initial",
		ContractStatus:   "accepted",
		ProposedBy:       "test",
		AcceptedBy:       "test",
		AcceptedAt:       "2026-03-27T00:00:00Z",
	}
	if err := WriteTaskContractCAS(path, contract, 0); err != nil {
		t.Fatalf("first contract write: %v", err)
	}
	written, err := LoadTaskContract(path)
	if err != nil {
		t.Fatalf("load first contract: %v", err)
	}
	if written.Revision != 1 {
		t.Fatalf("expected revision 1 after first contract write, got %+v", written)
	}

	contract.Objective = "updated"
	if err := WriteTaskContractCAS(path, contract, 1); err != nil {
		t.Fatalf("second contract write: %v", err)
	}
	written, err = LoadTaskContract(path)
	if err != nil {
		t.Fatalf("load second contract: %v", err)
	}
	if written.Revision != 2 || written.Objective != "updated" {
		t.Fatalf("expected revision 2 contract after second write, got %+v", written)
	}

	if err := WriteTaskContractCAS(path, contract, 1); !errors.Is(err, state.ErrCASConflict) {
		t.Fatalf("expected stale contract write to fail with CAS conflict, got %v", err)
	}
}

func TestWritePacketProgressCASIncrementsRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "packet-progress.json")
	progress := PacketProgress{
		SchemaVersion:    "kh.packet-progress.v1",
		Generator:        "test",
		UpdatedAt:        "2026-03-27T00:00:00Z",
		TaskID:           "T-1",
		ThreadKey:        "thread-1",
		PlanEpoch:        1,
		AcceptedPacketID: "packet_T-1_1",
	}
	if err := WritePacketProgressCAS(path, progress, 0); err != nil {
		t.Fatalf("first progress write: %v", err)
	}
	written, err := LoadPacketProgress(path)
	if err != nil {
		t.Fatalf("load first progress: %v", err)
	}
	if written.Revision != 1 {
		t.Fatalf("expected revision 1 after first progress write, got %+v", written)
	}

	progress.CompletedSliceIDs = []string{"T-1.slice.1"}
	if err := WritePacketProgressCAS(path, progress, 1); err != nil {
		t.Fatalf("second progress write: %v", err)
	}
	written, err = LoadPacketProgress(path)
	if err != nil {
		t.Fatalf("load second progress: %v", err)
	}
	if written.Revision != 2 || len(written.CompletedSliceIDs) != 1 {
		t.Fatalf("expected revision 2 progress after second write, got %+v", written)
	}

	if err := WritePacketProgressCAS(path, progress, 1); !errors.Is(err, state.ErrCASConflict) {
		t.Fatalf("expected stale progress write to fail with CAS conflict, got %v", err)
	}
}
