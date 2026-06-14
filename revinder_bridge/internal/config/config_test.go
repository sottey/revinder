package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefaultConfigFile(t *testing.T) {
	configPath := testConfigPath(t)
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.BindAddress != "*" {
		t.Fatalf("BindAddress = %q, want %q", cfg.BindAddress, "*")
	}
	if cfg.Port != 8080 {
		t.Fatalf("Port = %d, want %d", cfg.Port, 8080)
	}
	if filepath.Base(cfg.DatabasePath) != "revinder_bridge.sqlite" {
		t.Fatalf("DatabasePath = %q, want file named revinder_bridge.sqlite", cfg.DatabasePath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	want := "{\n  \"bind_address\": \"*\",\n  \"port\": 8080,\n  \"database_path\": \"revinder_bridge.sqlite\"\n}\n"
	if string(data) != want {
		t.Fatalf("config file = %q, want %q", string(data), want)
	}
}

func TestLoadConfigFile(t *testing.T) {
	configPath := testConfigPath(t)
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	if err := os.WriteFile(configPath, []byte(`{"bind_address":"127.0.0.1","port":9090}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.BindAddress != "127.0.0.1" {
		t.Fatalf("BindAddress = %q, want %q", cfg.BindAddress, "127.0.0.1")
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port = %d, want %d", cfg.Port, 9090)
	}
	if got, want := cfg.ServerAddress(), "127.0.0.1:9090"; got != want {
		t.Fatalf("ServerAddress() = %q, want %q", got, want)
	}
}

func TestLoadDefaultsEmptyConfigValues(t *testing.T) {
	configPath := testConfigPath(t)
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	if err := os.WriteFile(configPath, []byte(`{"bind_address":"","port":0,"database_path":""}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.BindAddress != "*" {
		t.Fatalf("BindAddress = %q, want %q", cfg.BindAddress, "*")
	}
	if cfg.Port != 8080 {
		t.Fatalf("Port = %d, want %d", cfg.Port, 8080)
	}
	if filepath.Base(cfg.DatabasePath) != "revinder_bridge.sqlite" {
		t.Fatalf("DatabasePath = %q, want file named revinder_bridge.sqlite", cfg.DatabasePath)
	}
}

func TestLoadRelativeDatabasePath(t *testing.T) {
	configPath := testConfigPath(t)
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	if err := os.WriteFile(configPath, []byte(`{"database_path":"data/tasks.sqlite"}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	exePath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(filepath.Dir(exePath), "data/tasks.sqlite")
	if cfg.DatabasePath != want {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, want)
	}
}

func TestLoadAbsoluteDatabasePath(t *testing.T) {
	configPath := testConfigPath(t)
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	dbPath := filepath.Join(t.TempDir(), "tasks.sqlite")
	if err := os.WriteFile(configPath, []byte(`{"database_path":"`+dbPath+`"}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DatabasePath != dbPath {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, dbPath)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	configPath := testConfigPath(t)
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	if err := os.WriteFile(configPath, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadInvalidPort(t *testing.T) {
	configPath := testConfigPath(t)
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	if err := os.WriteFile(configPath, []byte(`{"port":70000}`), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestServerAddressWildcard(t *testing.T) {
	cfg := Config{
		BindAddress: "*",
		Port:        8080,
	}

	if got, want := cfg.ServerAddress(), ":8080"; got != want {
		t.Fatalf("ServerAddress() = %q, want %q", got, want)
	}
}

func testConfigPath(t *testing.T) string {
	t.Helper()

	exePath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	return filepath.Join(filepath.Dir(exePath), "revinder_bridge.json")
}
