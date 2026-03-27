package main

import (
	"context"
	"testing"
	"time"

	"klein-harness/internal/runtime"
)

func TestDaemonLoopServiceStartAndStop(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	service := &daemonLoopService{
		root:     "/tmp/harness-root",
		interval: 5 * time.Second,
		options: runtime.RunOptions{
			WorkerID: "dashboard-daemon",
		},
		runLoop: func(ctx context.Context, root string, interval time.Duration, options runtime.RunOptions) error {
			if root != "/tmp/harness-root" {
				t.Fatalf("unexpected root: %q", root)
			}
			if interval != 5*time.Second {
				t.Fatalf("unexpected interval: %s", interval)
			}
			if options.WorkerID != "dashboard-daemon" {
				t.Fatalf("unexpected worker id: %q", options.WorkerID)
			}
			close(started)
			<-ctx.Done()
			close(stopped)
			return nil
		},
	}

	done := make(chan struct{})
	go func() {
		service.Start()
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("daemon service did not start")
	}

	service.Stop()

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("daemon service did not stop")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("daemon start did not return after stop")
	}
}
