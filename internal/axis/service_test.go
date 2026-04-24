package axis

import (
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
	config := newValidProductConfig()
	config.Runtime.Workdir = "../runtime"
	if _, err := WriteConfig(configPath, config); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
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
	if err := service.store.UpsertVersionRecord(MihomoVersionRecord{Version: "v1.19.21", Status: "installed", BinaryPath: binaryPath, InstalledAt: nowISO()}); err != nil {
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

func TestProductModelMutations(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	response, status := service.AddNodeSource(map[string]any{
		"name":     "osaka-fixed",
		"type":     "proxy",
		"enabled":  true,
		"protocol": "socks5",
		"endpoint": map[string]any{"server": "2.2.2.2", "port": 2080},
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddNodeSource() status = %d response = %#v", status, response)
	}

	response, status = service.AddRoute(map[string]any{
		"name":     "osaka-route",
		"enabled":  true,
		"strategy": "manual",
		"entry":    map[string]any{"source": "osaka-fixed"},
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddRoute() status = %d response = %#v", status, response)
	}

	response, status = service.UpdateUsage(map[string]any{
		"selectedRoute": "osaka-route",
		"localProxy":    map[string]any{"enabled": true, "type": "mixed", "listen": "127.0.0.1", "port": 7891},
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("UpdateUsage() status = %d response = %#v", status, response)
	}

	usage := service.GetUsage()
	if !usage.Ready || usage.SelectedRoute != "osaka-route" || usage.LocalProxy.Port != 7891 {
		t.Fatalf("unexpected usage view: %#v", usage)
	}
}

func TestPublicationRequiresAuth(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	response, status := service.AddPublication(map[string]any{
		"name":    "no-auth",
		"type":    "http_proxy",
		"enabled": true,
		"route":   "daily",
		"listen":  "0.0.0.0",
		"port":    18080,
	})
	if status == 200 || response["ok"] == true {
		t.Fatalf("expected publication auth validation failure, got %d %#v", status, response)
	}
}

func TestLocalNodeConnectionUsesDomainOnly(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	response, status := service.AddNodeSource(map[string]any{
		"name":     "home-node",
		"type":     "local_node",
		"enabled":  true,
		"protocol": "hysteria2",
		"local_node": map[string]any{
			"access_mode":   "bt_reverse_proxy",
			"listen":        "127.0.0.1",
			"port":          39014,
			"external_host": "axis.example.com",
			"external_port": 39014,
			"route":         "daily",
			"users":         []any{map[string]any{"username": "axis", "password": "secret"}},
		},
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddNodeSource() status = %d response = %#v", status, response)
	}
	connection, status := service.GetLocalNodeConnection("home-node")
	if status != 200 || !connection.OK || connection.Host != "axis.example.com" {
		t.Fatalf("unexpected connection response: %d %#v", status, connection)
	}
	if len(connection.Connections) != 1 || connection.Connections[0].Address != "axis.example.com:39014" {
		t.Fatalf("unexpected connection item: %#v", connection.Connections)
	}
	baota, status := service.GetLocalNodeBaotaConfig("home-node")
	if status != 200 || !baota.OK || !strings.Contains(baota.Snippet, "listen 39014 udp") {
		t.Fatalf("unexpected baota response: %d %#v", status, baota)
	}
}
