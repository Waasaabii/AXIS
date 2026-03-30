package axis

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestSelectGroupWithLandingProxyUsesSourceGroup(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	response, status := service.AddLandingProxy(map[string]any{
		"name":    "jp-egress",
		"type":    "socks5",
		"server":  "landing.example.com",
		"port":    443,
		"enabled": true,
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddLandingProxy() status = %d response = %#v", status, response)
	}

	response, status = service.UpdateEgressGroup("egress-hk-manual", map[string]any{
		"landing_proxy": "jp-egress",
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("UpdateEgressGroup() status = %d response = %#v", status, response)
	}

	service.state.Providers["airport-main"] = ProviderRecord{
		Provider: "airport-main",
		Nodes: []NodeInfo{
			{Name: "HK 01", Type: "ss", Server: "1.1.1.1", Port: 443},
		},
	}

	var runtimeGroupName string
	var runtimeProxyName string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runtimeGroupName = strings.TrimPrefix(r.URL.Path, "/proxies/")
		body, _ := io.ReadAll(r.Body)
		runtimeProxyName = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	service.controller = NewMihomoControllerAdapter(RuntimeConfig{
		ExternalController: server.URL,
		RenderOnly:         false,
	})

	result, status := service.SelectGroup("egress-hk-manual", "HK 01 -> jp-egress")
	if status != 200 {
		t.Fatalf("SelectGroup() status = %d result = %#v", status, result)
	}
	if runtimeGroupName != relaySourceGroupName("egress-hk-manual") {
		t.Fatalf("expected runtime group %q, got %q", relaySourceGroupName("egress-hk-manual"), runtimeGroupName)
	}
	if !strings.Contains(runtimeProxyName, `"name":"HK 01"`) {
		t.Fatalf("expected runtime proxy payload to target source node, got %s", runtimeProxyName)
	}
	if service.state.GroupSelections["egress-hk-manual"] != "HK 01 -> jp-egress" {
		t.Fatalf("unexpected stored selection: %q", service.state.GroupSelections["egress-hk-manual"])
	}
}
