package axis

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type MihomoControllerAdapter struct {
	baseURL    string
	secret     string
	renderOnly bool
	client     *http.Client
}

type controllerResult struct {
	OK      bool `json:"ok"`
	Status  int  `json:"status,omitempty"`
	Payload any  `json:"payload,omitempty"`
}

func NewMihomoControllerAdapter(runtimeConfig RuntimeConfig) *MihomoControllerAdapter {
	return &MihomoControllerAdapter{
		baseURL:    strings.TrimRight(firstNonEmpty(runtimeConfig.ExternalController, "http://127.0.0.1:9090"), "/"),
		secret:     runtimeConfig.ExternalSecret,
		renderOnly: runtimeConfig.RenderOnly,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (m *MihomoControllerAdapter) headers() http.Header {
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	if m.secret != "" {
		headers.Set("Authorization", "Bearer "+m.secret)
	}
	return headers
}

func (m *MihomoControllerAdapter) request(pathname, method string, body io.Reader) (*controllerResult, error) {
	req, err := http.NewRequest(method, m.baseURL+pathname, body)
	if err != nil {
		return nil, err
	}
	req.Header = m.headers()
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var payload any = strings.TrimSpace(string(raw))
	if len(raw) > 0 && json.Valid(raw) {
		var parsed any
		if err := json.Unmarshal(raw, &parsed); err == nil {
			payload = parsed
		}
	}

	return &controllerResult{
		OK:      resp.StatusCode >= 200 && resp.StatusCode < 300,
		Status:  resp.StatusCode,
		Payload: payload,
	}, nil
}

func (m *MihomoControllerAdapter) Probe() ControllerState {
	if m.renderOnly {
		return ControllerState{
			Reachable: false,
			Mode:      "render-only",
			Message:   "当前只保存配置，还没有接管代理核心。",
		}
	}

	result, err := m.request("/version", http.MethodGet, nil)
	if err != nil {
		return ControllerState{
			Reachable: false,
			Mode:      "managed",
			Message:   err.Error(),
		}
	}

	if !result.OK {
		return ControllerState{
			Reachable: false,
			Mode:      "managed",
			Message:   fmt.Sprintf("连接代理核心失败（状态码 %d）", result.Status),
		}
	}

	version := "unknown"
	switch payload := result.Payload.(type) {
	case string:
		version = payload
	case map[string]any:
		if value, ok := payload["version"].(string); ok && value != "" {
			version = value
		}
	}

	return ControllerState{
		Reachable: true,
		Mode:      "managed",
		Version:   version,
		Message:   "代理核心连接正常",
	}
}

func (m *MihomoControllerAdapter) RefreshProvider(providerName string) (*controllerResult, error) {
	if m.renderOnly {
		return &controllerResult{OK: false, Payload: map[string]any{"deferred": true, "message": "当前只保存配置，暂不刷新代理核心中的订阅。"}}, nil
	}
	return m.request("/providers/proxies/"+url.PathEscape(providerName), http.MethodPut, nil)
}

func (m *MihomoControllerAdapter) SelectProxy(groupName, proxyName string) (*controllerResult, error) {
	if m.renderOnly {
		return &controllerResult{OK: false, Payload: map[string]any{"deferred": true, "message": "当前只保存你的选择，暂不会切换代理核心。"}}, nil
	}
	body, _ := json.Marshal(map[string]string{"name": proxyName})
	return m.request("/proxies/"+url.PathEscape(groupName), http.MethodPut, strings.NewReader(string(body)))
}

func (m *MihomoControllerAdapter) ReloadConfig(configPath string) (*controllerResult, error) {
	if m.renderOnly {
		return &controllerResult{OK: false, Payload: map[string]any{"deferred": true, "message": "当前只生成配置文件，暂不会下发到代理核心。"}}, nil
	}
	body, _ := json.Marshal(map[string]string{"path": configPath})
	return m.request("/configs?force=true", http.MethodPut, strings.NewReader(string(body)))
}

func buildProvider(subscription Subscription) map[string]any {
	return map[string]any{
		"type":     "http",
		"url":      subscription.URL,
		"path":     fmt.Sprintf("providers/%s.yaml", subscription.Name),
		"interval": max(subscription.Interval, 3600),
		"health-check": map[string]any{
			"enable":          true,
			"url":             firstNonEmpty(subscription.HealthCheckURL, "https://www.gstatic.com/generate_204"),
			"interval":        max(subscription.HealthCheckInterval, 300),
			"timeout":         5000,
			"lazy":            true,
			"expected-status": 204,
		},
	}
}

func buildGroup(group EgressGroup) map[string]any {
	mode := firstNonEmpty(group.Mode, "manual")
	groupType := "select"
	if mode == "fallback" {
		groupType = "fallback"
	}
	if mode == "auto" {
		groupType = "url-test"
	}

	base := map[string]any{
		"name": group.Name,
		"type": groupType,
		"use":  []string{group.Provider},
	}
	if group.Filter != "" {
		base["filter"] = group.Filter
	}
	if group.ExcludeFilter != "" {
		base["exclude-filter"] = group.ExcludeFilter
	}
	if mode != "manual" {
		base["url"] = firstNonEmpty(group.HealthCheckURL, "https://www.gstatic.com/generate_204")
		base["interval"] = max(group.Interval, 300)
		base["lazy"] = true
		base["timeout"] = 5000
	}
	return base
}

func buildListener(listener Listener) map[string]any {
	return map[string]any{
		"name":   listener.Name,
		"type":   firstNonEmpty(listener.Type, "socks"),
		"listen": firstNonEmpty(listener.Listen, "0.0.0.0"),
		"port":   listener.Port,
		"udp":    listener.UDP,
		"users":  listener.Users,
		"proxy":  listener.EgressGroup,
	}
}

func RenderMihomoConfig(config *Config) (string, error) {
	enabledProviders := map[string]struct{}{}
	for _, subscription := range config.Subscriptions {
		if subscription.Enabled {
			enabledProviders[subscription.Name] = struct{}{}
		}
	}

	validGroups := map[string]struct{}{}
	groups := make([]map[string]any, 0, len(config.EgressGroups))
	for _, group := range config.EgressGroups {
		if _, ok := enabledProviders[group.Provider]; !ok {
			continue
		}
		validGroups[group.Name] = struct{}{}
		groups = append(groups, buildGroup(group))
	}

	providers := map[string]any{}
	for _, subscription := range config.Subscriptions {
		if !subscription.Enabled {
			continue
		}
		providers[subscription.Name] = buildProvider(subscription)
	}

	listeners := []map[string]any{}
	for _, listener := range config.Listeners {
		if !listener.Enabled {
			continue
		}
		if _, ok := validGroups[listener.EgressGroup]; !ok {
			continue
		}
		listeners = append(listeners, buildListener(listener))
	}

	document := map[string]any{
		"mode":                "rule",
		"log-level":           "info",
		"ipv6":                false,
		"external-controller": strings.TrimPrefix(strings.TrimPrefix(config.Runtime.ExternalController, "http://"), "https://"),
		"secret":              firstNonEmpty(config.Runtime.ExternalSecret, ""),
		"profile": map[string]any{
			"store-selected": true,
			"store-fake-ip":  false,
		},
		"proxy-providers": providers,
		"proxy-groups":    groups,
		"listeners":       listeners,
	}

	raw, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

func tryDecodeBase64Text(input string) string {
	normalized := strings.Join(strings.Fields(input), "")
	if len(normalized) < 16 {
		return ""
	}
	decoded, err := base64URLMaybeDecode(normalized)
	if err != nil {
		return ""
	}
	if strings.Contains(decoded, "://") || strings.Contains(decoded, "proxies:") {
		return decoded
	}
	return ""
}

func base64URLMaybeDecode(input string) (string, error) {
	if decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(input))); err == nil {
		return string(decoded), nil
	}
	decoded, err := io.ReadAll(base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(input)))
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func normalizeName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	decoded, err := url.QueryUnescape(name)
	if err != nil {
		return name
	}
	return decoded
}

func parseURLProtocol(protocol, line string) (NodeInfo, error) {
	parsed, err := url.Parse(line)
	if err != nil {
		return NodeInfo{}, err
	}
	port := 0
	fmt.Sscanf(parsed.Port(), "%d", &port)
	return NodeInfo{
		Type:    protocol,
		Name:    normalizeName(strings.TrimPrefix(parsed.Fragment, "#"), fmt.Sprintf("%s-%s:%s", protocol, parsed.Hostname(), parsed.Port())),
		Server:  parsed.Hostname(),
		Port:    port,
		Network: parsed.Query().Get("type"),
		TLS:     parsed.Query().Get("security"),
		Source:  line,
	}, nil
}

func parseVmess(line string) (NodeInfo, error) {
	raw := strings.TrimPrefix(line, "vmess://")
	decoded, err := base64URLMaybeDecode(raw)
	if err != nil {
		return NodeInfo{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
		return NodeInfo{}, err
	}
	port := 0
	fmt.Sscanf(fmt.Sprint(payload["port"]), "%d", &port)
	return NodeInfo{
		Type:    "vmess",
		Name:    normalizeName(fmt.Sprint(payload["ps"]), fmt.Sprintf("vmess-%v:%v", payload["add"], payload["port"])),
		Server:  fmt.Sprint(payload["add"]),
		Port:    port,
		Network: fmt.Sprint(payload["net"]),
		TLS:     fmt.Sprint(payload["tls"]),
		Source:  line,
	}, nil
}

func parseSS(line string) (NodeInfo, error) {
	parts := strings.SplitN(line, "#", 2)
	prefixPart := parts[0]
	fragment := ""
	if len(parts) == 2 {
		fragment = parts[1]
	}
	encoded := strings.TrimPrefix(prefixPart, "ss://")
	decoded := encoded
	if !strings.Contains(encoded, "@") {
		value, err := base64URLMaybeDecode(encoded)
		if err != nil {
			return NodeInfo{}, err
		}
		decoded = value
	}
	sections := strings.SplitN(decoded, "@", 2)
	if len(sections) != 2 {
		return NodeInfo{}, errors.New("invalid ss payload")
	}
	credentialPart := sections[0]
	serverPart := sections[1]
	credentialParts := strings.SplitN(credentialPart, ":", 2)
	serverSections := strings.SplitN(serverPart, ":", 2)
	port := 0
	if len(serverSections) == 2 {
		fmt.Sscanf(serverSections[1], "%d", &port)
	}
	cipher := ""
	if len(credentialParts) > 0 {
		cipher = credentialParts[0]
	}
	return NodeInfo{
		Type:   "shadowsocks",
		Name:   normalizeName(fragment, fmt.Sprintf("ss-%s:%d", serverSections[0], port)),
		Server: serverSections[0],
		Port:   port,
		TLS:    cipher,
		Source: line,
	}, nil
}

func parseClashYAML(text string) []NodeInfo {
	var parsed struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(text), &parsed); err != nil {
		return nil
	}
	nodes := make([]NodeInfo, 0, len(parsed.Proxies))
	for _, proxy := range parsed.Proxies {
		port := 0
		fmt.Sscanf(fmt.Sprint(proxy["port"]), "%d", &port)
		tls := ""
		if enabled, ok := proxy["tls"].(bool); ok && enabled {
			tls = "tls"
		}
		nodes = append(nodes, NodeInfo{
			Type:    fmt.Sprint(proxy["type"]),
			Name:    normalizeName(fmt.Sprint(proxy["name"]), fmt.Sprintf("%v:%v", proxy["server"], proxy["port"])),
			Server:  fmt.Sprint(proxy["server"]),
			Port:    port,
			Network: fmt.Sprint(proxy["network"]),
			TLS:     tls,
			Source:  "clash-yaml",
		})
	}
	return nodes
}

func ParseSubscriptionPayload(rawBody, sourceName string) []NodeInfo {
	trimmed := strings.TrimSpace(rawBody)
	if clashNodes := parseClashYAML(trimmed); len(clashNodes) > 0 {
		for index := range clashNodes {
			clashNodes[index].ID = createNodeID(sourceName, fmt.Sprintf("%s:%s:%s:%d", clashNodes[index].Type, clashNodes[index].Name, clashNodes[index].Server, clashNodes[index].Port))
		}
		return clashNodes
	}

	text := trimmed
	if decoded := tryDecodeBase64Text(trimmed); decoded != "" {
		text = decoded
	}

	lines := strings.Split(text, "\n")
	nodes := []NodeInfo{}
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var (
			node NodeInfo
			err  error
		)
		switch {
		case strings.HasPrefix(line, "vmess://"):
			node, err = parseVmess(line)
		case strings.HasPrefix(line, "vless://"):
			node, err = parseURLProtocol("vless", line)
		case strings.HasPrefix(line, "trojan://"):
			node, err = parseURLProtocol("trojan", line)
		case strings.HasPrefix(line, "hysteria2://"):
			node, err = parseURLProtocol("hysteria2", line)
		case strings.HasPrefix(line, "ss://"):
			node, err = parseSS(line)
		default:
			continue
		}
		if err != nil {
			continue
		}
		node.ID = createNodeID(sourceName, line)
		nodes = append(nodes, node)
	}
	return nodes
}

func maskURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path)
}

func userAgentForType(subscriptionType string) string {
	if subscriptionType == "clash-http" {
		return "clash-verge/v2.2.3"
	}
	return "clash-meta/v1.19.0"
}

func FetchProviderSnapshot(subscription Subscription) (*ProviderRecord, bool, int, string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, subscription.URL, nil)
	if err != nil {
		return nil, false, 0, "", err
	}
	req.Header.Set("User-Agent", userAgentForType(subscription.Type))

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, resp.StatusCode, resp.Status, err
	}

	nodes := ParseSubscriptionPayload(string(body), subscription.Name)
	record := &ProviderRecord{
		Provider:    subscription.Name,
		URLMasked:   maskURL(subscription.URL),
		RefreshedAt: nowISO(),
		NodeCount:   len(nodes),
		Nodes:       nodes,
	}

	return record, resp.StatusCode >= 200 && resp.StatusCode < 300, resp.StatusCode, resp.Status, nil
}

func defaultMihomoSupportMatrix() []MihomoVersionSupport {
	versions := []MihomoVersionSupport{
		{Version: "v1.19.21", Recommended: true, Supported: true, Compatibility: "当前版本与 AXIS 配合最稳定，推荐优先使用", ReleaseURL: "https://github.com/MetaCubeX/mihomo/releases/tag/v1.19.21"},
		{Version: "v1.19.20", Recommended: false, Supported: true, Compatibility: "已验证可以正常连接和使用", ReleaseURL: "https://github.com/MetaCubeX/mihomo/releases/tag/v1.19.20"},
		{Version: "v1.19.19", Recommended: false, Supported: true, Compatibility: "保守兼容版本，可用于回滚", ReleaseURL: "https://github.com/MetaCubeX/mihomo/releases/tag/v1.19.19"},
	}

	for index := range versions {
		assetName, assetURL, ok := buildMihomoAsset(versions[index].Version)
		versions[index].CurrentPlatform = ok
		if ok {
			versions[index].AssetName = assetName
			versions[index].AssetURL = assetURL
		} else {
			versions[index].Supported = false
			versions[index].Compatibility = "当前平台未配置受支持的官方制品命名"
		}
	}

	return versions
}

func buildMihomoAsset(version string) (string, string, bool) {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return "", "", false
	}

	goos := runtime.GOOS
	ext := ".gz"
	if goos == "windows" {
		ext = ".zip"
	}
	assetName := fmt.Sprintf("mihomo-%s-%s-%s%s", goos, arch, version, ext)
	return assetName, fmt.Sprintf("https://github.com/MetaCubeX/mihomo/releases/download/%s/%s", version, assetName), true
}

func downloadFile(targetPath, sourceURL string) error {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("下载失败: %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func extractGzipArchive(archivePath, binaryPath string) error {
	source, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer source.Close()

	reader, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return err
	}
	target, err := os.Create(binaryPath)
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err := io.Copy(target, reader); err != nil {
		return err
	}
	return os.Chmod(binaryPath, 0o755)
}
