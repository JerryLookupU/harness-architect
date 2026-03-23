package a2a

import (
	"path/filepath"
	"testing"
)

func TestAppendEventDedupesByKindAndIdempotencyKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	payload, err := NewPayload(map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	first, err := AppendEvent(path, Envelope{
		Kind:           "dispatch.issued",
		IdempotencyKey: "dispatch:T-1:1:1",
		From:           "orchestrator-node",
		To:             "worker-supervisor-node",
		Payload:        payload,
	})
	if err != nil {
		t.Fatalf("append first: %v", err)
	}
	second, err := AppendEvent(path, Envelope{
		Kind:           "dispatch.issued",
		IdempotencyKey: "dispatch:T-1:1:1",
		From:           "orchestrator-node",
		To:             "worker-supervisor-node",
		Payload:        payload,
	})
	if err != nil {
		t.Fatalf("append second: %v", err)
	}
	if second.Duplicate != true {
		t.Fatalf("expected duplicate append")
	}
	if first.Event.MessageID != second.Event.MessageID {
		t.Fatalf("expected reused message id")
	}
}
