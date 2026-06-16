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
  "memory_file": "/tmp/revinder-memory.jsonl",
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
	if cfg.MemoryFile != "/tmp/revinder-memory.jsonl" {
		t.Fatalf("MemoryFile = %q, want /tmp/revinder-memory.jsonl", cfg.MemoryFile)
	}
	if cfg.IntervalSeconds != 15 {
		t.Fatalf("IntervalSeconds = %d, want 15", cfg.IntervalSeconds)
	}
}

func TestEnvDefault(t *testing.T) {
	t.Setenv("REVINDER_MEMORY_TEST_VALUE", "configured")

	if got := envDefault("REVINDER_MEMORY_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("envDefault() = %q, want configured", got)
	}
	if got := envDefault("REVINDER_MEMORY_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("envDefault() = %q, want fallback", got)
	}
}
