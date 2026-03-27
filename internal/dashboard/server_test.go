package dashboard

import "testing"

func TestNewConfigParsesAddrAndAppliesDefaults(t *testing.T) {
	config, err := NewConfig("/tmp/harness-root", "127.0.0.1:7423")
	if err != nil {
		t.Fatalf("new config: %v", err)
	}
	if config.Root != "/tmp/harness-root" {
		t.Fatalf("unexpected root: %q", config.Root)
	}
	if config.Host != "127.0.0.1" {
		t.Fatalf("unexpected host: %q", config.Host)
	}
	if config.Port != 7423 {
		t.Fatalf("unexpected port: %d", config.Port)
	}
	if config.Name != "harness-dashboard" {
		t.Fatalf("unexpected service name: %q", config.Name)
	}
	if config.Timeout != 30000 {
		t.Fatalf("unexpected timeout: %d", config.Timeout)
	}
	if config.MaxBytes != 8<<20 {
		t.Fatalf("unexpected max bytes: %d", config.MaxBytes)
	}
}

func TestNewConfigRejectsInvalidAddr(t *testing.T) {
	if _, err := NewConfig("/tmp/harness-root", "7423"); err == nil {
		t.Fatal("expected invalid addr error")
	}
}
