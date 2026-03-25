package verify

import (
	"testing"

	"klein-harness/internal/dispatch"
)

func TestIngestVerificationEmitsFollowUp(t *testing.T) {
	root := t.TempDir()
	ticket, _, err := dispatch.Issue(dispatch.IssueRequest{
		Root:           root,
		TaskID:         "T-1",
		ThreadKey:      "thread-1",
		PlanEpoch:      1,
		Attempt:        1,
		IdempotencyKey: "dispatch:T-1:1:1",
		CausationID:    "route-1",
		WorkerClass:    "codex-go",
		Cwd:            root,
		Command:        "printf ok",
		PromptRef:      "prompts/worker-burst.md",
	})
	if err != nil {
		t.Fatalf("issue dispatch: %v", err)
	}
	result, err := Ingest(Request{
		Root:        root,
		TaskID:      "T-1",
		DispatchID:  ticket.DispatchID,
		PlanEpoch:   1,
		Attempt:     1,
		CausationID: "outcome-1",
		Status:      "failed",
		Summary:     "verification failed",
	})
	if err != nil {
		t.Fatalf("ingest verification: %v", err)
	}
	if result.FollowUpEvent != "replan.emitted" {
		t.Fatalf("expected replan follow up, got %s", result.FollowUpEvent)
	}
}

func TestIngestNoopRequiresEvidence(t *testing.T) {
	root := t.TempDir()
	ticket, _, err := dispatch.Issue(dispatch.IssueRequest{
		Root:           root,
		TaskID:         "T-1",
		ThreadKey:      "thread-1",
		PlanEpoch:      1,
		Attempt:        1,
		IdempotencyKey: "dispatch:T-1:1:1",
		CausationID:    "route-1",
		WorkerClass:    "codex-go",
		Cwd:            root,
		Command:        "printf ok",
		PromptRef:      "prompts/worker-burst.md",
	})
	if err != nil {
		t.Fatalf("issue dispatch: %v", err)
	}
	if _, err := Ingest(Request{
		Root:        root,
		TaskID:      "T-1",
		DispatchID:  ticket.DispatchID,
		PlanEpoch:   1,
		Attempt:     1,
		CausationID: "outcome-1",
		Status:      "already_satisfied",
		Summary:     "acceptance already satisfied",
	}); err != ErrNoopWithoutEvidence {
		t.Fatalf("expected noop evidence error, got %v", err)
	}
}
