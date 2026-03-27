package orchestration

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ConstraintSnapshot struct {
	SchemaVersion    string           `json:"schemaVersion"`
	Generator        string           `json:"generator"`
	GeneratedAt      string           `json:"generatedAt"`
	TaskID           string           `json:"taskId"`
	DispatchID       string           `json:"dispatchId"`
	PlanEpoch        int              `json:"planEpoch"`
	ConstraintSystem ConstraintSystem `json:"constraintSystem"`
	SoftRules        []ConstraintRule `json:"softRules"`
	HardRules        []ConstraintRule `json:"hardRules"`
}

func ConstraintSnapshotPath(root, taskID string) string {
	return filepath.Join(root, ".harness", "state", "constraints-"+taskID+".json")
}

func WriteConstraintSnapshot(path string, snapshot ConstraintSnapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}

func LoadConstraintSnapshot(path string) (ConstraintSnapshot, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return ConstraintSnapshot{}, err
	}
	var snapshot ConstraintSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return ConstraintSnapshot{}, err
	}
	return snapshot, nil
}

func SplitConstraintRules(system ConstraintSystem) (softRules, hardRules []ConstraintRule) {
	for _, rule := range system.Rules {
		switch rule.Enforcement {
		case "hard":
			hardRules = append(hardRules, rule)
		default:
			softRules = append(softRules, rule)
		}
	}
	return softRules, hardRules
}
