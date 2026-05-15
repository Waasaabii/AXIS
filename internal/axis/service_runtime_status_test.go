package axis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newRuntimeStatusTestService(t *testing.T, runtime RuntimeConfig) (*Service, func()) {
	t.Helper()
	rootDir, err := os.MkdirTemp(".", ".tmp-runtime-status-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	cleanup := func() {
		os.RemoveAll(rootDir)
	}

	configPath := filepath.Join(rootDir, "axis.yaml")
	if runtime.Workdir == "" {
		runtime.Workdir = ResolveDefaultRuntimeWorkdir(configPath)
	}
	if runtime.ExternalController == "" {
		runtime.ExternalController = "http://127.0.0.1:1"
	}
	if _, err := WriteConfig(configPath, &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 8787},
		Admin: AdminConfig{
			Username:        "admin",
			PasswordHash:    "hashed-password",
			SessionTTLHours: 12,
		},
		Runtime: runtime,
	}); err != nil {
		cleanup()
		t.Fatalf("WriteConfig() error = %v", err)
	}

	service, err := NewService(configPath)
	if err != nil {
		cleanup()
		t.Fatalf("NewService() error = %v", err)
	}
	return service, func() {
		service.Close()
		cleanup()
	}
}

func TestRuntimeStatusReportsMissingCoreWithoutBlockingBootstrap(t *testing.T) {
	service, cleanup := newRuntimeStatusTestService(t, RuntimeConfig{
		MihomoBinary:       "axis-missing-mihomo-core-test",
		ExternalController: "http://127.0.0.1:1",
		RenderOnly:         false,
	})
	defer cleanup()

	status := service.GetStatus()
	runtimeState := status["runtime"].(RuntimeState)
	if runtimeState.State != "core-missing" {
		t.Fatalf("expected core-missing runtime state, got %#v", runtimeState)
	}
	if runtimeState.Action != "manage-core" || !strings.Contains(runtimeState.Message, "代理核心程序") {
		t.Fatalf("expected actionable core missing message, got %#v", runtimeState)
	}

	bootstrap := service.GetBootstrapStatus(true, "admin")
	if !bootstrap.MainService.Ready {
		t.Fatalf("main service should stay reachable for in-app core management: %#v", bootstrap.MainService)
	}
	if bootstrap.MainService.State != "error" {
		t.Fatalf("expected visible main service error state, got %#v", bootstrap.MainService)
	}
	if bootstrap.NextStep != "dashboard" {
		t.Fatalf("expected bootstrap to allow dashboard, got %s", bootstrap.NextStep)
	}
}

func TestRuntimeStatusDistinguishesControllerUnreachable(t *testing.T) {
	service, cleanup := newRuntimeStatusTestService(t, RuntimeConfig{
		MihomoBinary:       "./mihomo",
		ExternalController: "http://127.0.0.1:1",
		RenderOnly:         false,
	})
	defer cleanup()

	binaryPath := filepath.Join(filepath.Dir(service.configPath), "mihomo")
	if err := os.WriteFile(binaryPath, []byte("# test binary placeholder\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	service.detectRuntime()
	service.detectController(false)

	if service.state.Runtime.State != "controller-unreachable" {
		t.Fatalf("expected controller-unreachable runtime state, got %#v", service.state.Runtime)
	}
	if service.state.Runtime.Action != "check-controller" {
		t.Fatalf("expected check-controller action, got %#v", service.state.Runtime)
	}
}

func TestRuntimeStatusReportsRenderOnly(t *testing.T) {
	service, cleanup := newRuntimeStatusTestService(t, RuntimeConfig{
		MihomoBinary:       "axis-missing-mihomo-core-test",
		ExternalController: "http://127.0.0.1:1",
		RenderOnly:         true,
	})
	defer cleanup()

	if service.state.Runtime.State != "render-only" {
		t.Fatalf("expected render-only runtime state, got %#v", service.state.Runtime)
	}
}
