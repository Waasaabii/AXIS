package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Waasaabii/AXIS/internal/axis"
)

func TestGetBootstrapStatusClearsDesktopSessionAfterReset(t *testing.T) {
	axisHome := t.TempDir()
	t.Setenv("AXIS_HOME", axisHome)

	configPath, err := axis.EnsureConfigPath("")
	if err != nil {
		t.Fatalf("EnsureConfigPath() error = %v", err)
	}
	if _, err := axis.ResetSetup(configPath); err != nil {
		t.Fatalf("ResetSetup() error = %v", err)
	}
	service, err := axis.NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	bindings := NewEngineBindings(service)
	if _, err := bindings.BootstrapAdmin(map[string]any{
		"username": "owner",
		"password": "secret123",
	}); err != nil {
		t.Fatalf("BootstrapAdmin() error = %v", err)
	}
	loginResult, err := bindings.Login(map[string]any{
		"username": "owner",
		"password": "secret123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if ok, _ := loginResult["ok"].(bool); !ok {
		t.Fatalf("unexpected login result: %#v", loginResult)
	}

	time.Sleep(20 * time.Millisecond)
	if _, err := axis.ResetSetup(configPath); err != nil {
		t.Fatalf("ResetSetup() error = %v", err)
	}
	if err := os.Chtimes(configPath, time.Now(), time.Now()); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	status, err := bindings.GetBootstrapStatus()
	if err != nil {
		t.Fatalf("GetBootstrapStatus() error = %v", err)
	}
	if status.Auth.Authenticated {
		t.Fatalf("expected session to be cleared after reset: %#v", status.Auth)
	}
	if status.NextStep != "setup" {
		t.Fatalf("expected next step to be setup, got %s", status.NextStep)
	}
	if !status.Setup.Required {
		t.Fatalf("expected setup to be required after reset: %#v", status.Setup)
	}

	if filepath.Dir(configPath) == "" {
		t.Fatal("config path should not be empty")
	}
}
