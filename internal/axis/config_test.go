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
		Server:         ServerConfig{Host: "127.0.0.1", Port: 9988},
		Admin:          AdminConfig{Username: "admin", Password: "updated-pass", SessionTTLHours: 12},
		Runtime:        RuntimeConfig{Workdir: "../runtime", MihomoBinary: "mihomo", ExternalController: "http://127.0.0.1:9090", RenderOnly: true},
		Subscriptions:  []Subscription{{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true, HealthCheckURL: "https://www.gstatic.com/generate_204", HealthCheckInterval: 300}},
		LandingProxies: []LandingProxy{{Name: "jp-egress", Type: "socks5", Server: "landing.example.com", Port: 443, Username: "relay", Password: "secret", Enabled: true}},
		EgressGroups:   []EgressGroup{{Name: "egress-hk", Provider: "airport-main", Mode: "manual", Filter: "(?i)港", LandingProxy: "jp-egress"}},
		Listeners:      []Listener{{Name: "hk-socks", Port: 10801, EgressGroup: "egress-hk", Enabled: true, UDP: true}},
	}

	if _, err := WriteConfig(configPath, config); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "port: 9988") || !strings.Contains(string(raw), "password: updated-pass") || !strings.Contains(string(raw), "landing_proxies:") || !strings.Contains(string(raw), "landing_proxy: jp-egress") {
		t.Fatalf("written config does not contain expected values:\n%s", string(raw))
	}
}

func TestValidateConfigRejectsUnknownLandingProxy(t *testing.T) {
	config := &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 8787},
		Admin:  AdminConfig{Username: "admin", Password: "admin", SessionTTLHours: 12},
		Runtime: RuntimeConfig{
			Workdir:            "../runtime",
			MihomoBinary:       "mihomo",
			ExternalController: "http://127.0.0.1:9090",
			RenderOnly:         true,
		},
		Subscriptions: []Subscription{{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true}},
		EgressGroups:  []EgressGroup{{Name: "egress-hk", Provider: "airport-main", Mode: "manual", LandingProxy: "missing-landing"}},
		Listeners:     []Listener{{Name: "hk-socks", Port: 10801, EgressGroup: "egress-hk", Enabled: true, UDP: true}},
	}

	if err := ValidateConfig(config); err == nil || !strings.Contains(err.Error(), "落地节点 missing-landing 不存在") {
		t.Fatalf("expected unknown landing proxy error, got %v", err)
	}
}
