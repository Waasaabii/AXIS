package axis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimePreflightSnapshot(t *testing.T) {
	tempDir := t.TempDir()
	runtimeDir := filepath.Join(tempDir, "runtime")
	providersDir := filepath.Join(runtimeDir, "providers")
	if err := os.MkdirAll(providersDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	for _, file := range []string{"mihomo.yaml", "mihomo.last-good.yaml", "control-state.json"} {
		if err := os.WriteFile(filepath.Join(runtimeDir, file), []byte("ok\n"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	snapshot := BuildRuntimePreflightSnapshot(
		filepath.Join(tempDir, "config", "proxyrelay.yaml"),
		&Config{Runtime: RuntimeConfig{RenderOnly: false, ExternalSecret: "secret"}},
		&RuntimeLayout{
			RuntimeDir:         runtimeDir,
			ProvidersDir:       providersDir,
			MihomoConfigPath:   filepath.Join(runtimeDir, "mihomo.yaml"),
			LastGoodConfigPath: filepath.Join(runtimeDir, "mihomo.last-good.yaml"),
			StatePath:          filepath.Join(runtimeDir, "control-state.json"),
		},
		&AppState{Runtime: RuntimeState{MihomoBinaryFound: true, MihomoBinary: "/usr/local/bin/mihomo", LastApplyStatus: "success", LastApplyMessage: "运行态已加载配置"}},
		&ControllerState{Reachable: true, Message: "controller 可达"},
	)

	if !snapshot.Ready {
		t.Fatalf("expected snapshot to be ready, got %#v", snapshot)
	}
	if snapshot.Status != "ok" {
		t.Fatalf("unexpected status: %s", snapshot.Status)
	}
}
