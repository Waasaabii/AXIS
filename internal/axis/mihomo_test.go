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
		Runtime: RuntimeConfig{ExternalController: "http://127.0.0.1:11235", ExternalSecret: "secret"},
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
