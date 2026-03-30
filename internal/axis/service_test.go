package axis

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, dir string) string {
	t.Helper()
	configDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(configDir, "proxyrelay.yaml")
	body := `
server:
  host: 127.0.0.1
  port: 8787
  startup_refresh: false
admin:
  username: admin
  password: admin
  password_hash: ""
  session_secret: test-secret
  session_ttl_hours: 12
runtime:
  workdir: ../runtime
  mihomo_binary: mihomo
  external_controller: http://127.0.0.1:9090
  external_secret: ""
  render_only: true
subscriptions:
  - name: airport-main
    type: mihomo-http
    url: https://example.com/sub
    interval: 3600
    enabled: true
    health_check_url: https://www.gstatic.com/generate_204
    health_check_interval: 300
egress_groups:
  - name: egress-hk-manual
    provider: airport-main
    mode: manual
    filter: "(?i)港|hk"
    exclude_filter: ""
listeners:
  - name: hk-socks
    type: socks
    listen: 0.0.0.0
    port: 10801
    udp: true
    enabled: true
    users:
      - username: user1
        password: pass1
    egress_group: egress-hk-manual
`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return configPath
}

func TestActivateMihomoVersion(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	versionDir := filepath.Join(service.layout.MihomoVersionsDir, "v1.19.21")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	binaryPath := filepath.Join(versionDir, "mihomo")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	record := MihomoVersionRecord{
		Version:     "v1.19.21",
		Status:      "installed",
		BinaryPath:  binaryPath,
		InstalledAt: nowISO(),
	}
	if err := service.store.UpsertVersionRecord(record); err != nil {
		t.Fatalf("UpsertVersionRecord() error = %v", err)
	}

	response, status := service.ActivateMihomoVersion("v1.19.21")
	if status != 200 {
		t.Fatalf("ActivateMihomoVersion() status = %d response = %#v", status, response)
	}
	if service.config.Runtime.MihomoBinary != binaryPath {
		t.Fatalf("expected config runtime binary to be updated, got %s", service.config.Runtime.MihomoBinary)
	}
}
