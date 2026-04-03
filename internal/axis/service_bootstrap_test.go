package axis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServiceBootstrapAdmin(t *testing.T) {
	rootDir, err := os.MkdirTemp(".", ".tmp-bootstrap-admin-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	defer os.RemoveAll(rootDir)

	configPath := filepath.Join(rootDir, "axis.yaml")
	if _, err := WriteConfig(configPath, &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 8787},
		Admin: AdminConfig{
			Username:        "admin",
			Password:        "",
			PasswordHash:    "",
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

	result, status := service.BootstrapAdmin("owner", "secret123")
	if status != 200 {
		t.Fatalf("BootstrapAdmin() status = %d, result = %#v", status, result)
	}

	setupState := service.GetSetupState()
	if setupState.NeedsPasswordReset {
		t.Fatalf("expected password reset requirement cleared: %#v", setupState)
	}
	if setupState.AdminUsername != "owner" {
		t.Fatalf("expected admin username to be updated, got %q", setupState.AdminUsername)
	}

	loginResult, loginStatus := service.Login("owner", "secret123")
	if loginStatus != 200 {
		t.Fatalf("Login() status = %d, result = %#v", loginStatus, loginResult)
	}

	failedLogin, failedStatus := service.Login("admin", "admin")
	if failedStatus != 401 {
		t.Fatalf("expected default credentials to be rejected, got %d %#v", failedStatus, failedLogin)
	}
}
