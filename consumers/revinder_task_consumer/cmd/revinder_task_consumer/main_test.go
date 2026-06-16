package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{
  "bridge_url": "http://127.0.0.1:9120",
  "token": "test-token",
  "target": "jsonl",
  "jsonl": {
    "path": "/tmp/revinder-tasks.jsonl"
  },
  "interval_seconds": 15
}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.BridgeURL != "http://127.0.0.1:9120" {
		t.Fatalf("BridgeURL = %q, want http://127.0.0.1:9120", cfg.BridgeURL)
	}
	if cfg.Token != "test-token" {
		t.Fatalf("Token = %q, want test-token", cfg.Token)
	}
	if cfg.Target != "jsonl" {
		t.Fatalf("Target = %q, want jsonl", cfg.Target)
	}
	if cfg.JSONL.Path != "/tmp/revinder-tasks.jsonl" {
		t.Fatalf("JSONL.Path = %q, want /tmp/revinder-tasks.jsonl", cfg.JSONL.Path)
	}
	if cfg.IntervalSeconds != 15 {
		t.Fatalf("IntervalSeconds = %d, want 15", cfg.IntervalSeconds)
	}
}

func TestLoadConfigReturnsInvalidJSONError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := loadConfig(path); err == nil {
		t.Fatal("loadConfig() error = nil, want error")
	}
}

func TestEnvDefault(t *testing.T) {
	t.Setenv("REVINDER_TEST_VALUE", "configured")

	if got := envDefault("REVINDER_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("envDefault() = %q, want configured", got)
	}
	if got := envDefault("REVINDER_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("envDefault() = %q, want fallback", got)
	}
}

func TestNewTaskProcessorRequiresJSONLPath(t *testing.T) {
	_, err := newTaskProcessor("jsonl", "")
	if err == nil || err.Error() != "jsonl.path is required when target is jsonl" {
		t.Fatalf("newTaskProcessor() error = %v, want jsonl path error", err)
	}
}

func TestNewTaskProcessorRejectsUnknownTarget(t *testing.T) {
	_, err := newTaskProcessor("unknown", "")
	if err == nil || err.Error() != `unknown target "unknown"` {
		t.Fatalf("newTaskProcessor() error = %v, want unknown target error", err)
	}
}
