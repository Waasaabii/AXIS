package axis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveConfigFromAPIRejectsAdminSecrets(t *testing.T) {
	rootDir := t.TempDir()
	configPath := filepath.Join(rootDir, "axis.yaml")

	if _, err := WriteConfig(configPath, &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 8787},
		Admin: AdminConfig{
			Username:        "admin",
			PasswordHash:    "hashed-password",
			SessionSecret:   "secret",
			SessionTTLHours: 12,
		},
		Runtime: RuntimeConfig{
			Workdir:            ResolveDefaultRuntimeWorkdir(configPath),
			MihomoBinary:       "mihomo",
			ExternalController: "http://127.0.0.1:9090",
			RenderOnly:         true,
		},
	}); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	next := mustJSONClone(*service.Config())
	next.Admin.PasswordHash = "evil"
	next.Admin.SessionSecret = "evil2"

	result, status := service.SaveConfigFromAPI(&next)
	if status != 400 {
		t.Fatalf("SaveConfigFromAPI() status = %d result = %#v", status, result)
	}
}

func TestSaveConfigFromAPIKeepsAdminSecrets(t *testing.T) {
	rootDir := t.TempDir()
	configPath := filepath.Join(rootDir, "axis.yaml")

	if _, err := WriteConfig(configPath, &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 8787},
		Admin: AdminConfig{
			Username:        "admin",
			PasswordHash:    "hashed-password",
			SessionSecret:   "secret",
			SessionTTLHours: 12,
		},
		Runtime: RuntimeConfig{
			Workdir:            ResolveDefaultRuntimeWorkdir(configPath),
			MihomoBinary:       "mihomo",
			ExternalController: "http://127.0.0.1:9090",
			RenderOnly:         true,
		},
	}); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	next := mustJSONClone(*service.Config())
	next.Admin.Password = ""
	next.Admin.PasswordHash = ""
	next.Admin.SessionSecret = ""
	next.Server.Port = 8788

	result, status := service.SaveConfigFromAPI(&next)
	if status != 200 {
		t.Fatalf("SaveConfigFromAPI() status = %d result = %#v", status, result)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.Server.Port != 8788 {
		t.Fatalf("expected server.port to be updated, got %d", loaded.Server.Port)
	}
	if loaded.Admin.PasswordHash != "hashed-password" {
		t.Fatalf("expected password hash to be preserved, got %q", loaded.Admin.PasswordHash)
	}
	if loaded.Admin.SessionSecret != "secret" {
		t.Fatalf("expected session secret to be preserved, got %q", loaded.Admin.SessionSecret)
	}
	if loaded.Admin.Password != "" {
		t.Fatalf("expected password to be empty, got %q", loaded.Admin.Password)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}
