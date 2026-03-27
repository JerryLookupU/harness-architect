package verify

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"klein-harness/internal/orchestration"
)

func recordCompletedExecutionSlice(root, taskID, dispatchID string) error {
	if taskID == "" || dispatchID == "" {
		return nil
	}
	packetPath := orchestration.AcceptedPacketPath(root, taskID)
	packet, err := orchestration.LoadAcceptedPacket(packetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	contractPath := orchestration.TaskContractPath(filepath.Join(root, ".harness", "artifacts", taskID, dispatchID))
	contract, err := orchestration.LoadTaskContract(contractPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if contract.ExecutionSliceID == "" {
		return nil
	}
	progressPath := orchestration.PacketProgressPath(root, taskID)
	progress, err := orchestration.LoadPacketProgress(progressPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if progress.TaskID == "" {
		progress = orchestration.PacketProgress{
			SchemaVersion:    "kh.packet-progress.v1",
			Generator:        "kh-orchestrator",
			TaskID:           taskID,
			ThreadKey:        packet.ThreadKey,
			PlanEpoch:        packet.PlanEpoch,
			AcceptedPacketID: packet.PacketID,
		}
	}
	progress.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	progress.LastDispatchID = dispatchID
	progress.CompletedSliceIDs = appendIfMissing(progress.CompletedSliceIDs, contract.ExecutionSliceID)
	return orchestration.WritePacketProgress(progressPath, progress)
}

func appendIfMissing(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}
