package orchestration

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type RejectedAlternative struct {
	CandidateID string `json:"candidateId"`
	Reason      string `json:"reason"`
}

type ExecutionTask struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Summary           string   `json:"summary"`
	InScope           []string `json:"inScope,omitempty"`
	DoneCriteria      []string `json:"doneCriteria,omitempty"`
	RequiredEvidence  []string `json:"requiredEvidence,omitempty"`
	VerificationSteps []string `json:"verificationSteps,omitempty"`
}

type AcceptedPacket struct {
	SchemaVersion        string                `json:"schemaVersion"`
	Generator            string                `json:"generator"`
	GeneratedAt          string                `json:"generatedAt"`
	TaskID               string                `json:"taskId"`
	ThreadKey            string                `json:"threadKey"`
	PlanEpoch            int                   `json:"planEpoch"`
	PacketID             string                `json:"packetId"`
	Objective            string                `json:"objective"`
	Constraints          []string              `json:"constraints"`
	FlowSelection        string                `json:"flowSelection"`
	PolicyTagsApplied    []string              `json:"policyTagsApplied,omitempty"`
	SelectedPlan         string                `json:"selectedPlan"`
	RejectedAlternatives []RejectedAlternative `json:"rejectedAlternatives,omitempty"`
	ExecutionTasks       []ExecutionTask       `json:"executionTasks"`
	VerificationPlan     map[string]any        `json:"verificationPlan"`
	DecisionRationale    string                `json:"decisionRationale"`
	OwnedPaths           []string              `json:"ownedPaths,omitempty"`
	TaskBudgets          map[string]any        `json:"taskBudgets,omitempty"`
	AcceptanceMarkers    []string              `json:"acceptanceMarkers,omitempty"`
	ReplanTriggers       []string              `json:"replanTriggers,omitempty"`
	RollbackHints        []string              `json:"rollbackHints,omitempty"`
	AcceptedAt           string                `json:"acceptedAt"`
	AcceptedBy           string                `json:"acceptedBy"`
}

type VerificationChecklistItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Required bool   `json:"required"`
	Status   string `json:"status,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type TaskContract struct {
	SchemaVersion         string                      `json:"schemaVersion"`
	Generator             string                      `json:"generator"`
	GeneratedAt           string                      `json:"generatedAt"`
	ContractID            string                      `json:"contractId"`
	TaskID                string                      `json:"taskId"`
	DispatchID            string                      `json:"dispatchId"`
	ThreadKey             string                      `json:"threadKey"`
	PlanEpoch             int                         `json:"planEpoch"`
	ExecutionSliceID      string                      `json:"executionSliceId"`
	Objective             string                      `json:"objective"`
	InScope               []string                    `json:"inScope,omitempty"`
	OutOfScope            []string                    `json:"outOfScope,omitempty"`
	DoneCriteria          []string                    `json:"doneCriteria,omitempty"`
	AcceptanceMarkers     []string                    `json:"acceptanceMarkers,omitempty"`
	VerificationChecklist []VerificationChecklistItem `json:"verificationChecklist,omitempty"`
	RequiredEvidence      []string                    `json:"requiredEvidence,omitempty"`
	ReviewRequired        bool                        `json:"reviewRequired"`
	ContractStatus        string                      `json:"contractStatus"`
	ProposedBy            string                      `json:"proposedBy"`
	AcceptedBy            string                      `json:"acceptedBy"`
	AcceptedAt            string                      `json:"acceptedAt"`
	AcceptedPacketPath    string                      `json:"acceptedPacketPath,omitempty"`
}

type PacketProgress struct {
	SchemaVersion     string   `json:"schemaVersion"`
	Generator         string   `json:"generator"`
	UpdatedAt         string   `json:"updatedAt"`
	TaskID            string   `json:"taskId"`
	ThreadKey         string   `json:"threadKey,omitempty"`
	PlanEpoch         int      `json:"planEpoch"`
	AcceptedPacketID  string   `json:"acceptedPacketId,omitempty"`
	CompletedSliceIDs []string `json:"completedSliceIds,omitempty"`
	LastDispatchID    string   `json:"lastDispatchId,omitempty"`
}

func AcceptedPacketPath(root, taskID string) string {
	return filepath.Join(root, ".harness", "state", "accepted-packet-"+taskID+".json")
}

func TaskContractPath(artifactDir string) string {
	return filepath.Join(artifactDir, "task-contract.json")
}

func PacketProgressPath(root, taskID string) string {
	return filepath.Join(root, ".harness", "state", "packet-progress-"+taskID+".json")
}

func WriteAcceptedPacket(path string, packet AcceptedPacket) error {
	return writeRuntimeObject(path, packet)
}

func LoadAcceptedPacket(path string) (AcceptedPacket, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return AcceptedPacket{}, err
	}
	var packet AcceptedPacket
	if err := json.Unmarshal(payload, &packet); err != nil {
		return AcceptedPacket{}, err
	}
	return packet, nil
}

func WriteTaskContract(path string, contract TaskContract) error {
	return writeRuntimeObject(path, contract)
}

func LoadTaskContract(path string) (TaskContract, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return TaskContract{}, err
	}
	var contract TaskContract
	if err := json.Unmarshal(payload, &contract); err != nil {
		return TaskContract{}, err
	}
	return contract, nil
}

func WritePacketProgress(path string, progress PacketProgress) error {
	return writeRuntimeObject(path, progress)
}

func LoadPacketProgress(path string) (PacketProgress, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return PacketProgress{}, err
	}
	var progress PacketProgress
	if err := json.Unmarshal(payload, &progress); err != nil {
		return PacketProgress{}, err
	}
	return progress, nil
}

func writeRuntimeObject(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}
