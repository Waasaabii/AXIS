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

func TestValidateConfigAcceptsManualProviderWithoutURL(t *testing.T) {
	config := &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 8787},
		Admin:  AdminConfig{Username: "admin", Password: "admin", SessionTTLHours: 12},
		Runtime: RuntimeConfig{
			Workdir:            "../runtime",
			MihomoBinary:       "mihomo",
			ExternalController: "http://127.0.0.1:9090",
			RenderOnly:         true,
		},
		Subscriptions: []Subscription{{Name: "tokyo-fixed", Type: "socks5", Server: "proxy.example.com", Port: 9001, Enabled: true}},
		EgressGroups:  []EgressGroup{{Name: "egress-fixed", Provider: "tokyo-fixed", Mode: "manual"}},
		Listeners:     []Listener{{Name: "fixed-socks", Port: 10801, EgressGroup: "egress-fixed", Enabled: true, UDP: true}},
	}

	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}

func TestValidateConfigRejectsFallbackGroupWithoutProxies(t *testing.T) {
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
		EgressGroups:  []EgressGroup{{Name: "egress-hk-fallback", Provider: "airport-main", Mode: "fallback"}},
		Listeners:     []Listener{{Name: "hk-socks", Port: 10801, EgressGroup: "egress-hk-fallback", Enabled: true, UDP: true}},
	}

	if err := ValidateConfig(config); err == nil || !strings.Contains(err.Error(), "顺序容灾组 egress-hk-fallback 至少需要选择一个节点") {
		t.Fatalf("expected fallback proxy validation error, got %v", err)
	}
}

func TestValidateConfigAcceptsTransitRouteAndTransitListener(t *testing.T) {
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
		EgressGroups:  []EgressGroup{{Name: "egress-hk", Provider: "airport-main", Mode: "manual"}},
		TransitRoutes: []TransitRoute{{
			Name:              "hk-transit",
			Enabled:           true,
			UpstreamProvider:  "airport-main",
			UpstreamProxyName: "HK 01",
			EgressGroup:       "egress-hk",
		}},
		Listeners: []Listener{{
			Name:         "transit-socks",
			Port:         10801,
			Enabled:      true,
			UDP:          true,
			RouteMode:    "transit",
			TransitRoute: "hk-transit",
		}},
	}

	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}

func TestValidateConfigRejectsUnknownTransitRoute(t *testing.T) {
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
		EgressGroups:  []EgressGroup{{Name: "egress-hk", Provider: "airport-main", Mode: "manual"}},
		Listeners: []Listener{{
			Name:         "transit-socks",
			Port:         10801,
			Enabled:      true,
			UDP:          true,
			RouteMode:    "transit",
			TransitRoute: "missing-route",
		}},
	}

	if err := ValidateConfig(config); err == nil || !strings.Contains(err.Error(), "绑定的中转线路 missing-route 不存在") {
		t.Fatalf("expected missing transit route error, got %v", err)
	}
}
