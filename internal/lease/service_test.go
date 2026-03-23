package lease

import (
	"testing"

	"klein-harness/internal/adapter"
	"klein-harness/internal/state"
)

func TestAcquireAndRecoverStaleLease(t *testing.T) {
	root := t.TempDir()
	record, err := Acquire(AcquireRequest{
		Root:        root,
		TaskID:      "T-1",
		DispatchID:  "dispatch-1",
		WorkerID:    "worker-1",
		LeaseID:     "lease-1",
		TTLSeconds:  -1,
		CausationID: "dispatch-1",
	})
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	if record.Status != "active" {
		t.Fatalf("expected active lease")
	}
	paths, err := adapter.Resolve(root)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	summary, err := loadSummary(paths.LeaseSummaryPath)
	if err != nil {
		t.Fatalf("load summary: %v", err)
	}
	expired := summary.Leases["lease-1"]
	expired.ExpiresAt = "1970-01-01T00:00:00Z"
	summary.Leases["lease-1"] = expired
	if _, err := state.WriteSnapshot(paths.LeaseSummaryPath, &summary, "test", summary.Revision); err != nil {
		t.Fatalf("write stale lease: %v", err)
	}
	recovered, err := RecoverStale(root, "recovery-1")
	if err != nil {
		t.Fatalf("recover stale: %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("expected one recovered lease, got %d", len(recovered))
	}
}
