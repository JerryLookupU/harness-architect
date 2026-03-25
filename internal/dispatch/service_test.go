package dispatch

import (
	"testing"
)

func TestIssueDedupesAndClaims(t *testing.T) {
	root := t.TempDir()
	ticket, duplicate, err := Issue(IssueRequest{
		Root:           root,
		TaskID:         "T-1",
		ThreadKey:      "thread-1",
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
		ThreadKey:      "thread-1",
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

func TestEnsureCurrentRejectsSupersededDispatch(t *testing.T) {
	root := t.TempDir()
	first, _, err := Issue(IssueRequest{
		Root:           root,
		TaskID:         "T-1",
		ThreadKey:      "thread-1",
		PlanEpoch:      2,
		Attempt:        1,
		IdempotencyKey: "dispatch:T-1:2:1",
		CausationID:    "route-1",
		WorkerClass:    "codex-go",
		Cwd:            root,
		Command:        "printf one",
		PromptRef:      "prompts/worker-burst.md",
	})
	if err != nil {
		t.Fatalf("issue first dispatch: %v", err)
	}
	if _, _, err := Issue(IssueRequest{
		Root:           root,
		TaskID:         "T-1",
		ThreadKey:      "thread-1",
		PlanEpoch:      2,
		Attempt:        2,
		IdempotencyKey: "dispatch:T-1:2:2",
		CausationID:    "route-2",
		WorkerClass:    "codex-go",
		Cwd:            root,
		Command:        "printf two",
		PromptRef:      "prompts/worker-burst.md",
	}); err != nil {
		t.Fatalf("issue second dispatch: %v", err)
	}
	if _, err := EnsureCurrent(root, first.DispatchID, first.TaskID, first.PlanEpoch); err == nil {
		t.Fatalf("expected superseded dispatch to be rejected")
	}
}
