package axis

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesAXISHomeOverride(t *testing.T) {
	axisHome := t.TempDir()
	t.Setenv("AXIS_HOME", axisHome)

	configPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}

	expected := filepath.Join(axisHome, "config", "proxyrelay.yaml")
	if configPath != expected {
		t.Fatalf("unexpected default config path: %s", configPath)
	}
}

func TestEnsureConfigPathCreatesDefaultConfig(t *testing.T) {
	axisHome := t.TempDir()
	t.Setenv("AXIS_HOME", axisHome)

	configPath, err := EnsureConfigPath("")
	if err != nil {
		t.Fatalf("EnsureConfigPath() error = %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.Admin.Password != "" {
		t.Fatalf("expected bootstrap password to be empty, got %q", config.Admin.Password)
	}
	if !config.Admin.RequiresPasswordReset {
		t.Fatalf("expected password reset requirement to be enabled")
	}
	if config.Runtime.Workdir != "../runtime" {
		t.Fatalf("unexpected runtime workdir: %s", config.Runtime.Workdir)
	}
}
