package axis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDefaultRuntimeWorkdir(t *testing.T) {
	if got := ResolveDefaultRuntimeWorkdir("/etc/proxyrelay/proxyrelay.yaml"); got != "/var/lib/proxyrelay/runtime" {
		t.Fatalf("unexpected system runtime dir: %s", got)
	}
	if got := ResolveDefaultRuntimeWorkdir("/opt/axis/config/proxyrelay.yaml"); got != "../runtime" {
		t.Fatalf("unexpected local runtime dir: %s", got)
	}
}

func TestWriteConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "proxyrelay.yaml")
	config := &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 9988},
		Admin: AdminConfig{Username: "admin", Password: "updated-pass", SessionTTLHours: 12},
		Runtime: RuntimeConfig{Workdir: "../runtime", MihomoBinary: "mihomo", ExternalController: "http://127.0.0.1:9090", RenderOnly: true},
		Subscriptions: []Subscription{{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true, HealthCheckURL: "https://www.gstatic.com/generate_204", HealthCheckInterval: 300}},
		EgressGroups:  []EgressGroup{{Name: "egress-hk", Provider: "airport-main", Mode: "manual", Filter: "(?i)港"}},
		Listeners:     []Listener{{Name: "hk-socks", Port: 10801, EgressGroup: "egress-hk", Enabled: true, UDP: true}},
	}

	if _, err := WriteConfig(configPath, config); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "port: 9988") || !strings.Contains(string(raw), "password: updated-pass") {
		t.Fatalf("written config does not contain expected values:\n%s", string(raw))
	}
}
