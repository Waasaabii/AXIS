package axis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createDynamicProxyTestService(listeners []Listener) *Service {
	return &Service{
		config: &Config{
			Subscriptions: []Subscription{
				{Name: "airport-main", URL: "https://example.com/sub", Type: "mihomo-http", Interval: 3600, Enabled: true},
			},
			EgressGroups: []EgressGroup{
				{Name: "egress-main", Provider: "airport-main", Mode: "manual"},
			},
			Listeners: listeners,
		},
		state: createEmptyState(),
	}
}

func TestDynamicProxyServiceReturnsTextProxyByDefault(t *testing.T) {
	service := createDynamicProxyTestService([]Listener{
		{
			Name:        "hk-socks",
			Type:        "socks",
			Listen:      "0.0.0.0",
			Port:        10801,
			Enabled:     true,
			Users:       []ListenerUser{{Username: "user_hk", Password: "pass_hk"}},
			RouteMode:   "direct",
			EgressGroup: "egress-main",
		},
	})

	request := httptest.NewRequest(http.MethodGet, "http://proxy.example.com:8787/api/proxy/next", nil)
	result := NewDynamicProxyService(service).GetNextProxy(request)
	if !result.OK || result.Format != "text" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if body, ok := result.Body.(string); !ok || body != "socks5://user_hk:pass_hk@proxy.example.com:10801" {
		t.Fatalf("unexpected proxy body: %#v", result.Body)
	}
}

func TestDynamicProxyServiceReturnsJSONAndProtocolFilter(t *testing.T) {
	service := createDynamicProxyTestService([]Listener{
		{
			Name:        "mixed-us",
			Type:        "mixed",
			Listen:      "0.0.0.0",
			Port:        10808,
			Enabled:     true,
			Users:       []ListenerUser{{Username: "mix_user", Password: "mix_pass"}},
			RouteMode:   "direct",
			EgressGroup: "egress-main",
		},
	})

	request := httptest.NewRequest(http.MethodGet, "http://proxy.example.com:8787/api/proxy/next?protocol=http&format=json", nil)
	result := NewDynamicProxyService(service).GetNextProxy(request)
	if !result.OK || result.Format != "json" {
		t.Fatalf("unexpected result: %#v", result)
	}
	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("unexpected body: %#v", result.Body)
	}
	data, _ := body["data"].(map[string]any)
	if data["proxy"] != "http://mix_user:mix_pass@proxy.example.com:10808" {
		t.Fatalf("unexpected proxy: %#v", data["proxy"])
	}
	if data["protocol"] != "http" {
		t.Fatalf("unexpected protocol: %#v", data["protocol"])
	}
}

func TestDynamicProxyServiceReturns503WhenNoAvailableProxy(t *testing.T) {
	service := createDynamicProxyTestService(nil)
	request := httptest.NewRequest(http.MethodGet, "http://proxy.example.com:8787/api/proxy/next", nil)
	result := NewDynamicProxyService(service).GetNextProxy(request)
	if result.OK || result.Status != http.StatusServiceUnavailable || result.Code != 50301 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestDynamicProxyServiceKeepsStickySession(t *testing.T) {
	service := createDynamicProxyTestService([]Listener{
		{
			Name:        "hk-socks",
			Type:        "socks",
			Listen:      "0.0.0.0",
			Port:        10801,
			Enabled:     true,
			Users:       []ListenerUser{{Username: "user1", Password: "pass1"}},
			RouteMode:   "direct",
			EgressGroup: "egress-main",
		},
		{
			Name:        "us-socks",
			Type:        "socks",
			Listen:      "0.0.0.0",
			Port:        10802,
			Enabled:     true,
			Users:       []ListenerUser{{Username: "user2", Password: "pass2"}},
			RouteMode:   "direct",
			EgressGroup: "egress-main",
		},
	})

	dynamicService := NewDynamicProxyService(service)
	first := dynamicService.GetNextProxy(httptest.NewRequest(http.MethodGet, "http://proxy.example.com:8787/api/proxy/next?session=sticky-1", nil))
	second := dynamicService.GetNextProxy(httptest.NewRequest(http.MethodGet, "http://proxy.example.com:8787/api/proxy/next?session=sticky-1", nil))
	if !first.OK || !second.OK || first.Body != second.Body {
		t.Fatalf("unexpected sticky results: %#v %#v", first, second)
	}
}

func TestDynamicProxyServiceHealthReportsPoolStatistics(t *testing.T) {
	service := createDynamicProxyTestService([]Listener{
		{
			Name:        "socks-a",
			Type:        "socks",
			Listen:      "0.0.0.0",
			Port:        10801,
			Enabled:     true,
			RouteMode:   "direct",
			EgressGroup: "egress-main",
		},
		{
			Name:        "mixed-b",
			Type:        "mixed",
			Listen:      "0.0.0.0",
			Port:        10802,
			Enabled:     false,
			RouteMode:   "direct",
			EgressGroup: "egress-main",
		},
	})

	health := NewDynamicProxyService(service).GetHealth(httptest.NewRequest(http.MethodGet, "http://proxy.example.com:8787/api/health", nil))
	pool, _ := health["pool"].(map[string]any)
	if health["status"] != "ok" || pool["total"] != 1 || pool["available"] != 1 || pool["disabled"] != 0 {
		t.Fatalf("unexpected health: %#v", health)
	}
}

func TestDynamicProxyServerEndpointBypassesSession(t *testing.T) {
	service := createDynamicProxyTestService([]Listener{
		{
			Name:        "hk-socks",
			Type:        "socks",
			Listen:      "0.0.0.0",
			Port:        10801,
			Enabled:     true,
			Users:       []ListenerUser{{Username: "user_hk", Password: "pass_hk"}},
			RouteMode:   "direct",
			EgressGroup: "egress-main",
		},
	})
	server := NewServer(service)
	request := httptest.NewRequest(http.MethodGet, "http://proxy.example.com:8787/api/proxy/next?format=json", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id header")
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["code"] != float64(0) {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}
