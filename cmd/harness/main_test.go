package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseDashboardOptionsEnablesDaemonByDefault(t *testing.T) {
	options, err := parseDashboardOptions(nil)
	if err != nil {
		t.Fatalf("parse dashboard options: %v", err)
	}
	if !options.EnableDaemon {
		t.Fatalf("expected dashboard to enable daemon loop by default")
	}
	if options.Addr != "127.0.0.1:7420" {
		t.Fatalf("unexpected default addr: %q", options.Addr)
	}
	if options.DaemonInterval != 30*time.Second {
		t.Fatalf("unexpected default daemon interval: %s", options.DaemonInterval)
	}
	if options.DaemonRunOptions.WorkerID != "dashboard-daemon" {
		t.Fatalf("unexpected default worker id: %q", options.DaemonRunOptions.WorkerID)
	}
}

func TestParseDashboardOptionsCanDisableDaemon(t *testing.T) {
	options, err := parseDashboardOptions([]string{"--no-daemon", "--addr", "127.0.0.1:9999", "--daemon-interval", "5s"})
	if err != nil {
		t.Fatalf("parse dashboard options: %v", err)
	}
	if options.EnableDaemon {
		t.Fatalf("expected --no-daemon to disable daemon loop")
	}
	if options.Addr != "127.0.0.1:9999" {
		t.Fatalf("unexpected addr: %q", options.Addr)
	}
	if options.DaemonInterval != 5*time.Second {
		t.Fatalf("unexpected daemon interval: %s", options.DaemonInterval)
	}
}

func TestRunDaemonRejectsRemovedRunOnce(t *testing.T) {
	err := runDaemon([]string{"run-once", "/tmp/repo"})
	if err == nil {
		t.Fatal("expected removed run-once error")
	}
	if !strings.Contains(err.Error(), "has been removed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
