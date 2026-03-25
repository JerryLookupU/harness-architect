package worker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"klein-harness/internal/dispatch"
)

func TestPrepareWritesDispatchTicketWorkerSpecAndPrompt(t *testing.T) {
	root := t.TempDir()
	harnessDir := filepath.Join(root, ".harness")
	if err := os.MkdirAll(filepath.Join(harnessDir, "verification-rules"), 0o755); err != nil {
		t.Fatalf("mkdir harness dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(harnessDir, "task-pool.json"), []byte(`{
  "tasks": [
    {
      "taskId": "T-1",
      "threadKey": "thread-1",
      "kind": "feature",
      "roleHint": "worker",
      "title": "Fix worker manifest plumbing",
      "summary": "Ensure worker reads a dispatch manifest before acting.",
      "workerMode": "execution",
      "planEpoch": 3,
      "ownedPaths": ["internal/worker/**"],
      "forbiddenPaths": [".harness/**"],
      "verificationRuleIds": ["VR-1"],
      "resumeStrategy": "resume",
      "preferredResumeSessionId": "sess-1",
      "routingModel": "gpt-5.4",
      "executionModel": "gpt-5.3-codex",
      "orchestrationSessionId": "orch-1",
      "promptStages": ["context_assembly", "plan", "execute", "verify"],
      "dispatch": {
        "worktreePath": ".worktrees/T-1",
        "branchName": "task/T-1",
        "diffBase": "refs/heads/main"
      }
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write task-pool: %v", err)
	}
	if err := os.WriteFile(filepath.Join(harnessDir, "project-meta.json"), []byte(`{
  "repoRole": "body_repo",
  "directTargetEditAllowed": false
}`), 0o644); err != nil {
		t.Fatalf("write project-meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(harnessDir, "verification-rules", "manifest.json"), []byte(`{
  "rules": [
    {
      "id": "VR-1",
      "title": "Go tests",
      "exec": "go test ./...",
      "timeout": 600,
      "readOnlySafe": true
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write verification manifest: %v", err)
	}

	bundle, err := Prepare(root, dispatch.Ticket{
		DispatchID:      "dispatch_T-1_3_1",
		IdempotencyKey:  "dispatch:T-1:epoch_3:attempt_1",
		TaskID:          "T-1",
		ThreadKey:       "thread-1",
		PlanEpoch:       3,
		Attempt:         1,
		PromptRef:       "prompts/worker-burst.md",
		ResumeSessionID: "sess-1",
	}, "lease-1")
	if err != nil {
		t.Fatalf("prepare bundle: %v", err)
	}

	var ticket struct {
		SchemaVersion           string `json:"schemaVersion"`
		DispatchID              string `json:"dispatchId"`
		IdempotencyKey          string `json:"idempotencyKey"`
		LeaseID                 string `json:"leaseId"`
		TaskID                  string `json:"taskId"`
		RepoRole                string `json:"repoRole"`
		DirectTargetEditAllowed bool   `json:"directTargetEditAllowed"`
		ArtifactDir             string `json:"artifactDir"`
		WorkerSpecPath          string `json:"workerSpecPath"`
		PacketSynthesis         struct {
			PlannerCount int `json:"plannerCount"`
			Judge        struct {
				ID string `json:"id"`
			} `json:"judge"`
		} `json:"packetSynthesis"`
		Verification struct {
			Commands []map[string]any `json:"commands"`
		} `json:"verification"`
	}
	payload, err := os.ReadFile(bundle.TicketPath)
	if err != nil {
		t.Fatalf("read ticket: %v", err)
	}
	if err := json.Unmarshal(payload, &ticket); err != nil {
		t.Fatalf("unmarshal ticket: %v", err)
	}
	if ticket.SchemaVersion != "kh.dispatch-ticket.v1" {
		t.Fatalf("unexpected ticket schema: %+v", ticket)
	}
	if ticket.DispatchID != "dispatch_T-1_3_1" || ticket.LeaseID != "lease-1" || ticket.TaskID != "T-1" {
		t.Fatalf("unexpected ticket identity: %+v", ticket)
	}
	if ticket.IdempotencyKey != "dispatch:T-1:epoch_3:attempt_1" {
		t.Fatalf("ticket missing idempotency key: %+v", ticket)
	}
	if ticket.RepoRole != "body_repo" || ticket.DirectTargetEditAllowed {
		t.Fatalf("project meta not propagated: %+v", ticket)
	}
	if ticket.ArtifactDir != bundle.ArtifactDir {
		t.Fatalf("artifact dir mismatch: ticket=%s bundle=%s", ticket.ArtifactDir, bundle.ArtifactDir)
	}
	if ticket.WorkerSpecPath != bundle.WorkerSpecPath {
		t.Fatalf("worker spec path mismatch: ticket=%s bundle=%s", ticket.WorkerSpecPath, bundle.WorkerSpecPath)
	}
	if len(ticket.Verification.Commands) != 1 {
		t.Fatalf("expected one verification command, got %d", len(ticket.Verification.Commands))
	}
	if ticket.PacketSynthesis.PlannerCount != 3 || ticket.PacketSynthesis.Judge.ID != "packet-judge" {
		t.Fatalf("packet synthesis contract missing: %+v", ticket.PacketSynthesis)
	}

	var workerSpec struct {
		SchemaVersion     string   `json:"schemaVersion"`
		DispatchID        string   `json:"dispatchId"`
		TaskID            string   `json:"taskId"`
		ThreadKey         string   `json:"threadKey"`
		PlanEpoch         int      `json:"planEpoch"`
		Objective         string   `json:"objective"`
		SelectedPlan      string   `json:"selectedPlan"`
		AcceptanceMarkers []string `json:"acceptanceMarkers"`
	}
	workerSpecPayload, err := os.ReadFile(bundle.WorkerSpecPath)
	if err != nil {
		t.Fatalf("read worker spec: %v", err)
	}
	if err := json.Unmarshal(workerSpecPayload, &workerSpec); err != nil {
		t.Fatalf("unmarshal worker spec: %v", err)
	}
	if workerSpec.SchemaVersion != "kh.worker-spec.v1" || workerSpec.DispatchID != ticket.DispatchID || workerSpec.TaskID != ticket.TaskID {
		t.Fatalf("worker spec identity mismatch: %+v", workerSpec)
	}
	if workerSpec.ThreadKey != "thread-1" || workerSpec.PlanEpoch != 3 {
		t.Fatalf("worker spec missing lineage: %+v", workerSpec)
	}
	if workerSpec.Objective == "" || workerSpec.SelectedPlan == "" || len(workerSpec.AcceptanceMarkers) != 1 {
		t.Fatalf("worker spec missing execution contract: %+v", workerSpec)
	}
	prompt, err := os.ReadFile(bundle.PromptPath)
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	promptText := string(prompt)
	if !strings.Contains(promptText, bundle.TicketPath) {
		t.Fatalf("prompt missing ticket path: %s", promptText)
	}
	if !strings.Contains(promptText, bundle.WorkerSpecPath) {
		t.Fatalf("prompt missing worker spec path: %s", promptText)
	}
	if !strings.Contains(promptText, "Final response:") {
		t.Fatalf("prompt missing worker close-out contract")
	}
	if !strings.Contains(promptText, "context assembly -> targeted research -> refine worker-spec understanding -> execute -> verify -> handoff") {
		t.Fatalf("prompt missing task-local loop")
	}
	if !strings.Contains(promptText, "task-local: 3 candidate worker-spec refinements, 1 judge") {
		t.Fatalf("prompt missing task-local b3e guidance")
	}
	if !strings.Contains(promptText, filepath.Join("prompts", "spec", "orchestrator.md")) {
		t.Fatalf("prompt missing orchestrator path")
	}
	if !strings.Contains(promptText, filepath.Join("prompts", "spec", "packet.md")) {
		t.Fatalf("prompt missing packet guide path")
	}
}
