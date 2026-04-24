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

func TestResolveRuntimeDir(t *testing.T) {
	configPath := "/opt/axis/config/proxyrelay.yaml"
	if got := ResolveRuntimeDir(configPath, "../runtime"); got != "/opt/axis/runtime" {
		t.Fatalf("unexpected relative runtime dir: %s", got)
	}
	if got := ResolveRuntimeDir(configPath, "/var/lib/proxyrelay/runtime"); got != "/var/lib/proxyrelay/runtime" {
		t.Fatalf("unexpected absolute runtime dir: %s", got)
	}
}

func newValidProductConfig() *Config {
	return &Config{
		Server:  ServerConfig{Host: "127.0.0.1", Port: 9988},
		Admin:   AdminConfig{Username: "admin", Password: "updated-pass", SessionTTLHours: 12},
		Runtime: RuntimeConfig{Workdir: "../runtime", MihomoBinary: "mihomo", ExternalController: "http://127.0.0.1:9090", RenderOnly: true},
		NodeSources: []NodeSource{{
			Name:         "airport-main",
			Type:         "subscription",
			Enabled:      true,
			Protocol:     "mihomo-http",
			Subscription: NodeSourceSubscription{URL: "https://sub.example.net/real", Interval: 3600},
		}},
		Routes: []RouteConfig{{
			Name:    "daily",
			Enabled: true,
			Entry:   RouteEndpointRef{Source: "airport-main"},
		}},
		Usage: UsageConfig{SelectedRoute: "daily", LocalProxy: LocalProxyUsage{Enabled: true, Type: "mixed", Listen: "127.0.0.1", Port: 7890}, VirtualInterface: VirtualInterfaceUsage{Mode: "system"}},
		Publications: []PublicationConfig{{
			Name:    "daily-lan",
			Type:    "http_proxy",
			Enabled: true,
			Route:   "daily",
			Listen:  "0.0.0.0",
			Port:    10801,
			Auth:    PublicationAuth{Username: "user", Password: "secret"},
		}},
	}
}

func TestWriteConfigUsesProductModel(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "proxyrelay.yaml")
	if _, err := WriteConfig(configPath, newValidProductConfig()); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, "node_sources:") || !strings.Contains(body, "routes:") || !strings.Contains(body, "publications:") {
		t.Fatalf("written config does not contain product model:\n%s", body)
	}
	if strings.Contains(body, "egress_groups:") || strings.Contains(body, "listeners:") {
		t.Fatalf("written config should not expose legacy fields:\n%s", body)
	}
}

func TestLoadConfigRejectsLegacyKeys(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "proxyrelay.yaml")
	if err := os.WriteFile(configPath, []byte("subscriptions: []\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := LoadConfig(configPath)
	if err == nil || !strings.Contains(err.Error(), "node_sources") {
		t.Fatalf("expected legacy config rejection, got %v", err)
	}
}

func TestValidateConfigRejectsMissingRouteSource(t *testing.T) {
	config := newValidProductConfig()
	config.Routes[0].Entry.Source = "missing"
	if err := ValidateConfig(config); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("expected missing source error, got %v", err)
	}
}

func TestValidateConfigRequiresPublicationAuth(t *testing.T) {
	config := newValidProductConfig()
	config.Publications[0].Auth = PublicationAuth{}
	if err := ValidateConfig(config); err == nil || !strings.Contains(err.Error(), "必须设置") {
		t.Fatalf("expected auth validation error, got %v", err)
	}
}

func TestValidateConfigAcceptsProxyNodeSource(t *testing.T) {
	config := newValidProductConfig()
	config.NodeSources = []NodeSource{{Name: "tokyo-fixed", Type: "proxy", Enabled: true, Protocol: "socks5", Endpoint: NodeSourceEndpoint{Server: "proxy.example.com", Port: 9001}}}
	config.Routes[0].Entry.Source = "tokyo-fixed"
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}

func TestValidateConfigAcceptsLocalNodeSource(t *testing.T) {
	config := newValidProductConfig()
	config.NodeSources = append(config.NodeSources, NodeSource{Name: "home-node", Type: "local_node", Enabled: true, Protocol: "trojan", LocalNode: LocalNodeConfig{AccessMode: "bt_reverse_proxy", Listen: "127.0.0.1", Port: 39013, ExternalHost: "axis.example.com", ExternalPort: 443, Route: "daily", Users: []ListenerUser{{Username: "axis", Password: "secret"}}}})
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}
