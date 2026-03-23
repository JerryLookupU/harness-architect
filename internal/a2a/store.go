package a2a

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type Envelope struct {
	SchemaVersion string          `json:"schemaVersion"`
	MessageID     string          `json:"messageId"`
	Kind          string          `json:"kind"`
	IdempotencyKey string         `json:"idempotencyKey"`
	TraceID       string          `json:"traceId,omitempty"`
	CausationID   string          `json:"causationId,omitempty"`
	From          string          `json:"from"`
	To            string          `json:"to"`
	CreatedAt     string          `json:"createdAt"`
	RequestID     string          `json:"requestId,omitempty"`
	TaskID        string          `json:"taskId,omitempty"`
	PlanEpoch     int             `json:"planEpoch,omitempty"`
	Attempt       int             `json:"attempt,omitempty"`
	SessionID     string          `json:"sessionId,omitempty"`
	WorkerID      string          `json:"workerId,omitempty"`
	LeaseID       string          `json:"leaseId,omitempty"`
	ReasonCodes   []string        `json:"reasonCodes,omitempty"`
	Payload       json.RawMessage `json:"payload"`
}

type AppendResult struct {
	Event     Envelope
	Duplicate bool
}

func NewPayload(value any) (json.RawMessage, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func LoadEvents(path string) ([]Envelope, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	events := make([]Envelope, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event Envelope
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}

func AppendEvent(path string, event Envelope) (AppendResult, error) {
	events, err := LoadEvents(path)
	if err != nil {
		return AppendResult{}, err
	}
	for _, existing := range events {
		if existing.Kind == event.Kind && existing.IdempotencyKey == event.IdempotencyKey {
			return AppendResult{Event: existing, Duplicate: true}, nil
		}
	}
	if event.SchemaVersion == "" {
		event.SchemaVersion = "a2a.v1"
	}
	if event.CreatedAt == "" {
		event.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if event.MessageID == "" {
		event.MessageID = fmt.Sprintf("msg_%d_%06d", time.Now().UTC().UnixNano(), rand.Intn(1_000_000))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return AppendResult{}, err
	}
	handle, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return AppendResult{}, err
	}
	defer handle.Close()
	line, err := json.Marshal(event)
	if err != nil {
		return AppendResult{}, err
	}
	if _, err := handle.Write(append(line, '\n')); err != nil {
		return AppendResult{}, err
	}
	return AppendResult{Event: event}, nil
}
