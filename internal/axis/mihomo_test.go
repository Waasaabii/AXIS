package axis

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReloadConfig(t *testing.T) {
	var requestPath string
	var requestMethod string
	var authHeader string
	var requestBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.RequestURI()
		requestMethod = r.Method
		authHeader = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		requestBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"reloaded"}`))
	}))
	defer server.Close()

	adapter := NewMihomoControllerAdapter(RuntimeConfig{
		ExternalController: server.URL,
		ExternalSecret:     "controller-secret",
		RenderOnly:         false,
	})

	result, err := adapter.ReloadConfig("/var/lib/proxyrelay/runtime/mihomo.yaml")
	if err != nil {
		t.Fatalf("ReloadConfig() error = %v", err)
	}
	if !result.OK {
		t.Fatalf("expected reload to succeed")
	}
	if requestPath != "/configs?force=true" || requestMethod != http.MethodPut {
		t.Fatalf("unexpected request: %s %s", requestMethod, requestPath)
	}
	if authHeader != "Bearer controller-secret" {
		t.Fatalf("unexpected auth header: %s", authHeader)
	}
	if !strings.Contains(requestBody, "/var/lib/proxyrelay/runtime/mihomo.yaml") {
		t.Fatalf("unexpected request body: %s", requestBody)
	}
}

func TestRenderMihomoConfig(t *testing.T) {
	output, err := RenderMihomoConfig(&Config{
		Runtime:       RuntimeConfig{ExternalController: "http://127.0.0.1:11235", ExternalSecret: "secret"},
		Subscriptions: []Subscription{{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true, HealthCheckURL: "https://www.gstatic.com/generate_204", HealthCheckInterval: 300}},
		EgressGroups:  []EgressGroup{{Name: "egress-hk-manual", Provider: "airport-main", Mode: "manual", Filter: "(?i)港"}},
		Listeners:     []Listener{{Name: "hk-socks", Type: "socks", Listen: "0.0.0.0", Port: 10801, UDP: true, Enabled: true, EgressGroup: "egress-hk-manual"}},
	})
	if err != nil {
		t.Fatalf("RenderMihomoConfig() error = %v", err)
	}
	if !strings.Contains(output, "proxy-providers:") || !strings.Contains(output, "listeners:") || !strings.Contains(output, "hk-socks") {
		t.Fatalf("unexpected rendered config:\n%s", output)
	}
}

func TestRenderMihomoConfigWithLandingProxy(t *testing.T) {
	output, err := RenderMihomoConfig(&Config{
		Runtime: RuntimeConfig{ExternalController: "http://127.0.0.1:11235", ExternalSecret: "secret"},
		Subscriptions: []Subscription{
			{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true, HealthCheckURL: "https://www.gstatic.com/generate_204", HealthCheckInterval: 300},
		},
		LandingProxies: []LandingProxy{
			{Name: "jp-egress", Type: "socks5", Server: "landing.example.com", Port: 443, Username: "relay", Password: "secret", Enabled: true},
		},
		EgressGroups: []EgressGroup{
			{Name: "egress-hk-manual", Provider: "airport-main", Mode: "manual", Filter: "(?i)港", LandingProxy: "jp-egress"},
		},
		Listeners: []Listener{
			{Name: "hk-socks", Type: "socks", Listen: "0.0.0.0", Port: 10801, UDP: true, Enabled: true, EgressGroup: "egress-hk-manual"},
		},
	})
	if err != nil {
		t.Fatalf("RenderMihomoConfig() error = %v", err)
	}
	for _, expected := range []string{
		"proxies:",
		"name: landing::jp-egress",
		"type: relay",
		"name: egress-hk-manual::source",
		"name: egress-hk-manual",
		"landing::jp-egress",
		"proxy: egress-hk-manual",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("rendered config missing %q:\n%s", expected, output)
		}
	}
}

func TestRenderMihomoConfigWithManualProvider(t *testing.T) {
	output, err := RenderMihomoConfig(&Config{
		Runtime: RuntimeConfig{ExternalController: "http://127.0.0.1:11235", ExternalSecret: "secret"},
		Subscriptions: []Subscription{
			{Name: "tokyo-fixed", Type: "socks5", Server: "proxy.example.com", Port: 9001, Enabled: true},
		},
		EgressGroups: []EgressGroup{
			{Name: "egress-fixed", Provider: "tokyo-fixed", Mode: "manual"},
		},
		Listeners: []Listener{
			{Name: "fixed-socks", Type: "socks", Listen: "0.0.0.0", Port: 10801, UDP: true, Enabled: true, EgressGroup: "egress-fixed"},
		},
	})
	if err != nil {
		t.Fatalf("RenderMihomoConfig() error = %v", err)
	}
	for _, expected := range []string{
		"proxy-providers:",
		"type: file",
		"path: providers/tokyo-fixed.yaml",
		"name: egress-fixed",
		"proxy: egress-fixed",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("rendered config missing %q:\n%s", expected, output)
		}
	}
}

func TestRenderMihomoConfigWithFallbackGroup(t *testing.T) {
	output, err := RenderMihomoConfig(&Config{
		Runtime: RuntimeConfig{ExternalController: "http://127.0.0.1:11235", ExternalSecret: "secret"},
		Subscriptions: []Subscription{
			{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true},
		},
		EgressGroups: []EgressGroup{
			{Name: "egress-hk-fallback", Provider: "airport-main", Mode: "fallback", Proxies: []string{"HK 01", "HK 02"}, Interval: 60},
		},
		Listeners: []Listener{
			{Name: "hk-socks", Type: "socks", Listen: "0.0.0.0", Port: 10801, UDP: true, Enabled: true, EgressGroup: "egress-hk-fallback"},
		},
	})
	if err != nil {
		t.Fatalf("RenderMihomoConfig() error = %v", err)
	}
	for _, expected := range []string{
		"type: fallback",
		"__axis_fb_",
		"filter: ^HK 01$",
		"filter: ^HK 02$",
		"proxy: egress-hk-fallback",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("rendered fallback config missing %q:\n%s", expected, output)
		}
	}
}

func TestRenderMihomoConfigWithTransitRoute(t *testing.T) {
	output, err := RenderMihomoConfig(&Config{
		Runtime: RuntimeConfig{ExternalController: "http://127.0.0.1:11235", ExternalSecret: "secret"},
		Subscriptions: []Subscription{
			{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true, HealthCheckURL: "https://www.gstatic.com/generate_204", HealthCheckInterval: 300},
		},
		EgressGroups: []EgressGroup{
			{Name: "egress-hk-manual", Provider: "airport-main", Mode: "manual", Filter: "(?i)港"},
		},
		TransitRoutes: []TransitRoute{
			{Name: "hk-transit", Enabled: true, UpstreamProvider: "airport-main", UpstreamProxyName: "HK 01", EgressGroup: "egress-hk-manual"},
		},
		Listeners: []Listener{
			{Name: "transit-socks", Type: "socks", Listen: "0.0.0.0", Port: 10801, UDP: true, Enabled: true, RouteMode: "transit", TransitRoute: "hk-transit"},
		},
	})
	if err != nil {
		t.Fatalf("RenderMihomoConfig() error = %v", err)
	}
	for _, expected := range []string{
		"__axis_tr_up_",
		"__axis_tr_pvd_",
		"__axis_tr_grp_",
		"dialer-proxy",
		"filter: ^HK 01$",
		"proxy: __axis_tr_grp_",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("rendered transit config missing %q:\n%s", expected, output)
		}
	}
}

func TestRenderMihomoConfigWithHysteria2LocalNodeListener(t *testing.T) {
	config := &Config{
		Runtime:       RuntimeConfig{ExternalController: "http://127.0.0.1:11235", ExternalSecret: "secret"},
		Subscriptions: []Subscription{{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true}},
		EgressGroups:  []EgressGroup{{Name: "daily", Provider: "airport-main", Mode: "manual"}},
		Listeners:     []Listener{{Name: "hy2-node", Type: "hysteria2", Listen: "0.0.0.0", Port: 39014, UDP: true, Enabled: true, Users: []ListenerUser{{Username: "axis", Password: "secret"}}, Certificate: "fullchain.pem", PrivateKey: "privkey.pem", EgressGroup: "daily"}},
	}
	output, err := RenderMihomoConfig(config)
	if err != nil {
		t.Fatalf("RenderMihomoConfig() error = %v", err)
	}
	for _, expected := range []string{"type: hysteria2", "users:", "axis: secret", "certificate: fullchain.pem", "private-key: privkey.pem"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("rendered config missing %q:\n%s", expected, output)
		}
	}
}

func TestManualProviderFileSupportsTrojanAndHysteria2(t *testing.T) {
	for _, item := range []Subscription{
		{Name: "trojan-node", Type: "trojan", Server: "axis.example.com", Port: 39013, Password: "secret", TLS: true, SNI: "axis.example.com"},
		{Name: "hy2-node", Type: "hysteria2", Server: "axis.example.com", Port: 39014, Password: "secret", TLS: true, SNI: "axis.example.com"},
	} {
		content, err := buildManualProviderFileContent(item)
		if err != nil {
			t.Fatalf("buildManualProviderFileContent() error = %v", err)
		}
		output := string(content)
		for _, expected := range []string{"type: " + item.Type, "server: axis.example.com", "password: secret", "sni: axis.example.com"} {
			if !strings.Contains(output, expected) {
				t.Fatalf("manual provider missing %q:\n%s", expected, output)
			}
		}
	}
}
