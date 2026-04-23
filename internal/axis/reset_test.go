package axis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResetSetup(t *testing.T) {
	axisHome := t.TempDir()
	t.Setenv("AXIS_HOME", axisHome)
	configDir := filepath.Join(axisHome, "config")
	runtimeDir := filepath.Join(axisHome, "runtime")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(runtimeDir, "providers"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	configPath := filepath.Join(configDir, "proxyrelay.yaml")
	config := &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 8787},
		Admin: AdminConfig{
			Username:        "admin",
			PasswordHash:    "hashed-password",
			SessionSecret:   "secret",
			SessionTTLHours: 12,
		},
		Runtime: RuntimeConfig{
			Workdir:            "../runtime",
			MihomoBinary:       "mihomo",
			ExternalController: "http://127.0.0.1:9090",
			ExternalSecret:     "controller-secret",
			RenderOnly:         false,
		},
		Subscriptions: []Subscription{{Name: "airport-main", URL: "https://sub.example.net/real", Type: "mihomo-http", Interval: 3600, Enabled: true}},
		EgressGroups:  []EgressGroup{{Name: "egress-hk", Provider: "airport-main", Mode: "manual"}},
		Listeners:     []Listener{{Name: "hk-socks", Port: 10801, EgressGroup: "egress-hk", Enabled: true, UDP: true}},
	}
	if _, err := WriteConfig(configPath, config); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "axis.db"), []byte("db"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "providers", "sample.yaml"), []byte("provider"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := ResetSetup(configPath)
	if err != nil {
		t.Fatalf("ResetSetup() error = %v", err)
	}
	resolvedConfigPath := mustResolveRealPath(t, configPath)
	if mustResolveRealPath(t, result.ConfigPath) != resolvedConfigPath {
		t.Fatalf("unexpected config path: %s", result.ConfigPath)
	}
	resolvedRuntimeDir := mustResolveRealPath(t, runtimeDir)
	if mustResolveRealPath(t, result.RuntimeDir) != resolvedRuntimeDir {
		t.Fatalf("unexpected runtime dir: %s", result.RuntimeDir)
	}

	resetConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if resetConfig.Admin.Password != "" {
		t.Fatalf("expected bootstrap password to be empty, got %q", resetConfig.Admin.Password)
	}
	if !resetConfig.Admin.RequiresPasswordReset {
		t.Fatalf("expected password reset requirement to be enabled")
	}
	if resetConfig.Admin.PasswordHash != "" {
		t.Fatalf("expected password hash to be cleared, got %q", resetConfig.Admin.PasswordHash)
	}
	if resetConfig.Admin.SessionSecret != "" {
		t.Fatalf("expected session secret to be cleared, got %q", resetConfig.Admin.SessionSecret)
	}
	if len(resetConfig.Subscriptions) != 0 || len(resetConfig.EgressGroups) != 0 || len(resetConfig.Listeners) != 0 {
		t.Fatalf("expected setup resources to be cleared: %#v", resetConfig)
	}
	if resetConfig.Runtime.ExternalController != "http://127.0.0.1:9090" || resetConfig.Runtime.RenderOnly {
		t.Fatalf("expected runtime settings to be preserved: %#v", resetConfig.Runtime)
	}

	entries, err := os.ReadDir(runtimeDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected runtime dir to be empty, got %d entries", len(entries))
	}
}

func TestResetSetupRejectsDangerousRuntimeDir(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config", "proxyrelay.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	config := BuildDefaultConfig(configPath)
	config.Runtime.Workdir = "/"
	if _, err := WriteConfig(configPath, config); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	if _, err := ResetSetup(configPath); err == nil {
		t.Fatal("expected dangerous runtime dir to be rejected")
	}
}

func mustResolveRealPath(t *testing.T, value string) string {
	t.Helper()
	resolved, err := filepath.Abs(value)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	if evaled, err := filepath.EvalSymlinks(resolved); err == nil {
		return evaled
	}
	return resolved
}
