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

func TestAddManualSubscriptionGeneratesProviderFileAndSnapshot(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	response, status := service.AddSubscription(map[string]any{
		"type":        "socks5",
		"import_text": "socks5://207.211.181.215:45001:user:pass",
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddSubscription() status = %d response = %#v", status, response)
	}

	record := service.getProviderRecord("socks5-207.211.181.215-45001")
	if record.NodeCount != 1 || len(record.Nodes) != 1 {
		t.Fatalf("unexpected provider snapshot: %#v", record)
	}
	if record.URLMasked != "socks5://207.211.181.215:45001" {
		t.Fatalf("unexpected masked endpoint: %s", record.URLMasked)
	}

	filePath := filepath.Join(service.layout.ProvidersDir, "socks5-207.211.181.215-45001.yaml")
	raw, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "server: 207.211.181.215") || !strings.Contains(string(raw), "password: pass") {
		t.Fatalf("unexpected manual provider file:\n%s", string(raw))
	}
}

func TestUpdateManualSubscriptionRenamesAndSyncsReferences(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	response, status := service.AddSubscription(map[string]any{
		"name":     "tokyo-fixed",
		"type":     "socks5",
		"server":   "1.1.1.1",
		"port":     1080,
		"username": "user",
		"password": "pass",
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddSubscription() status = %d response = %#v", status, response)
	}

	response, status = service.AddEgressGroup(map[string]any{
		"name":     "egress-fixed",
		"provider": "tokyo-fixed",
		"mode":     "manual",
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddEgressGroup() status = %d response = %#v", status, response)
	}

	response, status = service.AddTransitRoute(map[string]any{
		"name":                "fixed-transit",
		"upstream_provider":   "tokyo-fixed",
		"upstream_proxy_name": "tokyo-fixed",
		"egress_group":        "egress-fixed",
		"enabled":             true,
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddTransitRoute() status = %d response = %#v", status, response)
	}
	service.state.GroupSelections["egress-fixed"] = "tokyo-fixed"

	response, status = service.UpdateSubscription("tokyo-fixed", map[string]any{
		"name":     "osaka-fixed",
		"server":   "2.2.2.2",
		"port":     2080,
		"username": "next-user",
		"password": "next-pass",
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("UpdateSubscription() status = %d response = %#v", status, response)
	}

	if service.config.Subscriptions[1].Name != "osaka-fixed" || service.config.Subscriptions[1].Server != "2.2.2.2" || service.config.Subscriptions[1].Port != 2080 {
		t.Fatalf("unexpected updated subscription: %#v", service.config.Subscriptions[1])
	}
	if service.config.EgressGroups[1].Provider != "osaka-fixed" {
		t.Fatalf("expected egress group provider to be renamed, got %#v", service.config.EgressGroups[1])
	}
	if service.state.GroupSelections["egress-fixed"] != "osaka-fixed" {
		t.Fatalf("expected group selection to be renamed, got %q", service.state.GroupSelections["egress-fixed"])
	}
	if service.config.TransitRoutes[0].UpstreamProvider != "osaka-fixed" || service.config.TransitRoutes[0].UpstreamProxyName != "osaka-fixed" {
		t.Fatalf("unexpected transit route after rename: %#v", service.config.TransitRoutes[0])
	}

	if _, exists := service.state.Providers["tokyo-fixed"]; exists {
		t.Fatalf("old provider state should be removed: %#v", service.state.Providers["tokyo-fixed"])
	}
	record := service.getProviderRecord("osaka-fixed")
	if record.NodeCount != 1 || len(record.Nodes) != 1 || record.Nodes[0].Name != "osaka-fixed" {
		t.Fatalf("unexpected renamed provider snapshot: %#v", record)
	}

	oldFile := filepath.Join(service.layout.ProvidersDir, "tokyo-fixed.yaml")
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("expected old provider file to be removed, stat err = %v", err)
	}
	newFile := filepath.Join(service.layout.ProvidersDir, "osaka-fixed.yaml")
	raw, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "server: 2.2.2.2") || !strings.Contains(string(raw), "username: next-user") {
		t.Fatalf("unexpected renamed manual provider file:\n%s", string(raw))
	}
}

func TestUpdateSubscriptionRejectsRemoteProvider(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	response, status := service.UpdateSubscription("airport-main", map[string]any{
		"name": "airport-2",
	})
	if status != 400 {
		t.Fatalf("UpdateSubscription() status = %d response = %#v", status, response)
	}
	if !strings.Contains(stringValue(response["error"]), "只支持编辑手动节点") {
		t.Fatalf("unexpected error: %#v", response)
	}
}

func TestAddFallbackGroupStoresProxyOrder(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	response, status := service.AddEgressGroup(map[string]any{
		"name":     "egress-hk-fallback",
		"provider": "airport-main",
		"mode":     "fallback",
		"proxies":  []any{"HK 01", "HK 02", "HK 01"},
		"interval": 60,
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddEgressGroup() status = %d response = %#v", status, response)
	}

	var saved *EgressGroup
	for index := range service.config.EgressGroups {
		if service.config.EgressGroups[index].Name == "egress-hk-fallback" {
			saved = &service.config.EgressGroups[index]
			break
		}
	}
	if saved == nil {
		t.Fatal("fallback group not found in config")
	}
	if len(saved.Proxies) != 2 || saved.Proxies[0] != "HK 01" || saved.Proxies[1] != "HK 02" {
		t.Fatalf("unexpected fallback proxy order: %#v", saved.Proxies)
	}
	if service.state.GroupSelections["egress-hk-fallback"] != "HK 01" {
		t.Fatalf("unexpected stored fallback selection: %q", service.state.GroupSelections["egress-hk-fallback"])
	}
}

func TestAddTransitRouteAndTransitListener(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	service.state.Providers["airport-main"] = ProviderRecord{
		Provider: "airport-main",
		Nodes: []NodeInfo{
			{Name: "HK 01", Type: "ss", Server: "1.1.1.1", Port: 443},
			{Name: "HK 02", Type: "ss", Server: "1.1.1.2", Port: 443},
		},
	}
	service.state.GroupSelections["egress-hk-manual"] = "HK 02"

	response, status := service.AddTransitRoute(map[string]any{
		"name":                "hk-transit",
		"upstream_provider":   "airport-main",
		"upstream_proxy_name": "HK 01",
		"egress_group":        "egress-hk-manual",
		"enabled":             true,
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddTransitRoute() status = %d response = %#v", status, response)
	}
	service.state.Providers["airport-main"] = ProviderRecord{
		Provider: "airport-main",
		Nodes: []NodeInfo{
			{Name: "HK 01", Type: "ss", Server: "1.1.1.1", Port: 443},
			{Name: "HK 02", Type: "ss", Server: "1.1.1.2", Port: 443},
		},
	}
	service.state.GroupSelections["egress-hk-manual"] = "HK 02"

	response, status = service.AddListener(map[string]any{
		"name":          "transit-socks",
		"port":          10802,
		"route_mode":    "transit",
		"transit_route": "hk-transit",
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddListener() status = %d response = %#v", status, response)
	}
	service.state.Providers["airport-main"] = ProviderRecord{
		Provider: "airport-main",
		Nodes: []NodeInfo{
			{Name: "HK 01", Type: "ss", Server: "1.1.1.1", Port: 443},
			{Name: "HK 02", Type: "ss", Server: "1.1.1.2", Port: 443},
		},
	}
	service.state.GroupSelections["egress-hk-manual"] = "HK 02"

	routes := service.GetTransitRoutes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 transit route, got %d", len(routes))
	}
	if routes[0].Status != "configured" || routes[0].CurrentProxy != "HK 02" {
		t.Fatalf("unexpected transit route view: %#v", routes[0])
	}

	listeners := service.GetListeners()
	if len(listeners) != 2 {
		t.Fatalf("expected 2 listeners, got %d", len(listeners))
	}
	var transitListener *ListenerView
	for index := range listeners {
		if listeners[index].Name == "transit-socks" {
			transitListener = &listeners[index]
			break
		}
	}
	if transitListener == nil {
		t.Fatal("transit listener not found")
	}
	if transitListener.RouteMode != "transit" || transitListener.TransitRoute != "hk-transit" {
		t.Fatalf("unexpected transit listener: %#v", transitListener)
	}
	if !strings.Contains(transitListener.RouteSummary, "HK 01 -> HK 02") {
		t.Fatalf("unexpected transit route summary: %s", transitListener.RouteSummary)
	}
}

func TestRunTransitRouteHealthcheck(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeTestConfig(t, tempDir)
	service, err := NewService(configPath)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	service.state.Providers["airport-main"] = ProviderRecord{
		Provider: "airport-main",
		Nodes: []NodeInfo{
			{Name: "HK 01", Type: "ss", Server: "1.1.1.1", Port: 443},
		},
	}

	response, status := service.AddTransitRoute(map[string]any{
		"name":                "hk-transit",
		"upstream_provider":   "airport-main",
		"upstream_proxy_name": "HK 01",
		"egress_group":        "egress-hk-manual",
		"enabled":             true,
	})
	if status != 200 || response["ok"] != true {
		t.Fatalf("AddTransitRoute() status = %d response = %#v", status, response)
	}

	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"delay":123}`))
	}))
	defer server.Close()

	service.controller = NewMihomoControllerAdapter(RuntimeConfig{
		ExternalController: server.URL,
		RenderOnly:         false,
	})

	result, status := service.RunTransitRouteHealthcheck("hk-transit")
	if status != 200 {
		t.Fatalf("RunTransitRouteHealthcheck() status = %d result = %#v", status, result)
	}
	if !strings.Contains(requestPath, "/group/"+buildTransitMirrorGroupName("hk-transit")+"/delay") {
		t.Fatalf("unexpected healthcheck request path: %s", requestPath)
	}
	if service.state.TransitRoutes["hk-transit"].LastTestStatus != "success" || service.state.TransitRoutes["hk-transit"].LastTestDelay != 123 {
		t.Fatalf("unexpected transit state: %#v", service.state.TransitRoutes["hk-transit"])
	}
}
