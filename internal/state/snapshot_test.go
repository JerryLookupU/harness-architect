package state

import (
	"errors"
	"path/filepath"
	"testing"
)

type snapshotFixture struct {
	Metadata Metadata `json:"-"`
	Name     string   `json:"name"`
}

func TestWriteSnapshotCAS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	doc := &snapshotFixture{Name: "first"}
	meta, err := WriteSnapshot(path, doc, "test", 0)
	if err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	if meta.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", meta.Revision)
	}

	doc.Name = "second"
	meta, err = WriteSnapshot(path, doc, "test", 1)
	if err != nil {
		t.Fatalf("rewrite snapshot: %v", err)
	}
	if meta.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", meta.Revision)
	}

	if _, err := WriteSnapshot(path, doc, "test", 1); !errors.Is(err, ErrCASConflict) {
		t.Fatalf("expected cas conflict, got %v", err)
	}
}
