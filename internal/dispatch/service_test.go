package dispatch

import (
	"testing"
)

func TestIssueDedupesAndClaims(t *testing.T) {
	root := t.TempDir()
	ticket, duplicate, err := Issue(IssueRequest{
		Root:           root,
		TaskID:         "T-1",
		PlanEpoch:      2,
		Attempt:        1,
		IdempotencyKey: "dispatch:T-1:2:1",
		CausationID:    "route-1",
		WorkerClass:    "codex-go",
		Cwd:            root,
		Command:        "printf test",
		PromptRef:      "prompts/worker-burst.md",
	})
	if err != nil {
		t.Fatalf("issue dispatch: %v", err)
	}
	if duplicate {
		t.Fatalf("expected fresh issue")
	}
	second, duplicate, err := Issue(IssueRequest{
		Root:           root,
		TaskID:         "T-1",
		PlanEpoch:      2,
		Attempt:        1,
		IdempotencyKey: "dispatch:T-1:2:1",
		CausationID:    "route-1",
		WorkerClass:    "codex-go",
		Cwd:            root,
		Command:        "printf test",
		PromptRef:      "prompts/worker-burst.md",
	})
	if err != nil {
		t.Fatalf("issue duplicate dispatch: %v", err)
	}
	if !duplicate || second.DispatchID != ticket.DispatchID {
		t.Fatalf("expected deduped dispatch")
	}

	claimed, err := Claim(ClaimRequest{
		Root:        root,
		DispatchID:  ticket.DispatchID,
		WorkerID:    "worker-1",
		LeaseID:     "lease-1",
		CausationID: "claim-1",
	})
	if err != nil {
		t.Fatalf("claim dispatch: %v", err)
	}
	if claimed.Status != "claimed" {
		t.Fatalf("expected claimed status, got %s", claimed.Status)
	}
}
