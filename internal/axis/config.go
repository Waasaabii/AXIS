package axis

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultLocalRuntimeWorkdir  = "../runtime"
	defaultSystemRuntimeWorkdir = "/var/lib/proxyrelay/runtime"
)

func ResolveDefaultRuntimeWorkdir(configPath string) string {
	configDir := filepath.Clean(filepath.Dir(configPath))
	if configDir == "/etc/proxyrelay" {
		return defaultSystemRuntimeWorkdir
	}
	return defaultLocalRuntimeWorkdir
}

func ResolveRuntimeDir(configPath, workdir string) string {
	if filepath.IsAbs(workdir) {
		return filepath.Clean(workdir)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(configPath), workdir))
}

func LoadConfig(configPath string) (*Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if containsLegacyConfigKeys(raw) {
		return nil, errors.New("当前配置仍使用旧结构，请改用 node_sources、routes、usage、publications 新配置")
	}

	var parsed Config
	if len(raw) > 0 {
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
	}

	config := normalizeNewConfig(parsed, configPath)
	deriveLegacyRuntimeConfig(config)
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

func containsLegacyConfigKeys(raw []byte) bool {
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return false
	}
	for _, key := range []string{"subscriptions", "landing_proxies", "egress_groups", "transit_routes", "listeners"} {
		if _, ok := doc[key]; ok {
			return true
		}
	}
	return false
}

func normalizeNewConfig(parsed Config, configPath string) *Config {
	config := &Config{
		Server: ServerConfig{
			Host:           firstNonEmpty(parsed.Server.Host, "127.0.0.1"),
			Port:           max(parsed.Server.Port, 8787),
			StartupRefresh: parsed.Server.StartupRefresh,
		},
		Admin: AdminConfig{
			Username:              firstNonEmpty(parsed.Admin.Username, "admin"),
			Password:              parsed.Admin.Password,
			PasswordHash:          parsed.Admin.PasswordHash,
			SessionSecret:         parsed.Admin.SessionSecret,
			SessionTTLHours:       max(parsed.Admin.SessionTTLHours, 12),
			RequiresPasswordReset: parsed.Admin.PasswordHash == "" && parsed.Admin.Password == "",
		},
		Runtime: RuntimeConfig{
			Workdir:            firstNonEmpty(parsed.Runtime.Workdir, ResolveDefaultRuntimeWorkdir(configPath)),
			MihomoBinary:       firstNonEmpty(parsed.Runtime.MihomoBinary, "mihomo"),
			ExternalController: firstNonEmpty(parsed.Runtime.ExternalController, "http://127.0.0.1:9090"),
			ExternalSecret:     parsed.Runtime.ExternalSecret,
			RenderOnly:         true,
		},
		NodeSources:  parsed.NodeSources,
		Routes:       parsed.Routes,
		Usage:        parsed.Usage,
		Publications: parsed.Publications,
		LLM: LLMConfig{
			Enabled:        parsed.LLM.Enabled,
			BaseURL:        strings.TrimRight(strings.TrimSpace(parsed.LLM.BaseURL), "/"),
			APIKey:         parsed.LLM.APIKey,
			Model:          strings.TrimSpace(parsed.LLM.Model),
			Endpoint:       firstNonEmpty(strings.TrimSpace(parsed.LLM.Endpoint), "responses"),
			TimeoutSeconds: max(parsed.LLM.TimeoutSeconds, 60),
		},
	}
	if parsed.Runtime.Workdir != "" || parsed.Runtime.MihomoBinary != "" || parsed.Runtime.ExternalController != "" || parsed.Runtime.ExternalSecret != "" || parsed.Runtime.RenderOnly {
		config.Runtime.RenderOnly = parsed.Runtime.RenderOnly
	}
	for index := range config.NodeSources {
		if config.NodeSources[index].Type == "" {
			config.NodeSources[index].Type = "subscription"
		}
		if config.NodeSources[index].Protocol == "" {
			config.NodeSources[index].Protocol = defaultNodeSourceProtocol(config.NodeSources[index])
		}
		if config.NodeSources[index].Type == "local_node" {
			if config.NodeSources[index].LocalNode.AccessMode == "" {
				config.NodeSources[index].LocalNode.AccessMode = "bt_reverse_proxy"
			}
			if config.NodeSources[index].LocalNode.Listen == "" {
				config.NodeSources[index].LocalNode.Listen = "127.0.0.1"
			}
			if config.NodeSources[index].LocalNode.ExternalPort == 0 {
				config.NodeSources[index].LocalNode.ExternalPort = defaultLocalNodeExternalPort(config.NodeSources[index].Protocol, config.NodeSources[index].LocalNode.Port)
			}
			if config.NodeSources[index].LocalNode.SNI == "" {
				config.NodeSources[index].LocalNode.SNI = config.NodeSources[index].LocalNode.ExternalHost
			}
		}
		if config.NodeSources[index].Subscription.Interval == 0 {
			config.NodeSources[index].Subscription.Interval = 3600
		}
		if config.NodeSources[index].Subscription.HealthCheckURL == "" {
			config.NodeSources[index].Subscription.HealthCheckURL = "https://www.gstatic.com/generate_204"
		}
		if config.NodeSources[index].Subscription.HealthCheckInterval == 0 {
			config.NodeSources[index].Subscription.HealthCheckInterval = 300
		}
	}
	for index := range config.Routes {
		if config.Routes[index].Strategy == "" {
			config.Routes[index].Strategy = "manual"
		}
		if config.Routes[index].HealthCheck.URL == "" {
			config.Routes[index].HealthCheck.URL = "https://www.gstatic.com/generate_204"
		}
		if config.Routes[index].HealthCheck.Interval == 0 {
			config.Routes[index].HealthCheck.Interval = 300
		}
	}
	if config.Usage.LocalProxy.Type == "" {
		config.Usage.LocalProxy.Type = "mixed"
	}
	if config.Usage.LocalProxy.Listen == "" {
		config.Usage.LocalProxy.Listen = "127.0.0.1"
	}
	if config.Usage.LocalProxy.Port == 0 {
		config.Usage.LocalProxy.Port = 7890
	}
	if config.Usage.VirtualInterface.Mode == "" {
		config.Usage.VirtualInterface.Mode = "system"
	}
	for index := range config.Publications {
		if config.Publications[index].Listen == "" {
			config.Publications[index].Listen = "0.0.0.0"
		}
		if config.Publications[index].AccessScope == "" {
			config.Publications[index].AccessScope = "lan"
		}
		if config.Publications[index].Format == "" {
			config.Publications[index].Format = "url"
		}
	}
	return config
}

func defaultNodeSourceProtocol(source NodeSource) string {
	switch source.Type {
	case "subscription":
		return "mihomo-http"
	case "axis":
		return "axis"
	case "local_node":
		return "http"
	default:
		return "socks5"
	}
}

func defaultLocalNodeExternalPort(protocol string, port int) int {
	switch protocol {
	case "trojan":
		return 443
	case "hysteria2":
		return 39014
	default:
		return port
	}
}

func ValidateConfig(config *Config) error {
	if config.Server.Port <= 0 {
		return errors.New("server.port 必须大于 0")
	}
	if config.Admin.Username == "" {
		return errors.New("admin.username 不能为空")
	}
	if config.LLM.Enabled {
		if strings.TrimSpace(config.LLM.BaseURL) == "" {
			return errors.New("llm.base_url 不能为空")
		}
		if strings.TrimSpace(config.LLM.APIKey) == "" {
			return errors.New("llm.api_key 不能为空")
		}
	}

	sourceNames := map[string]NodeSource{}
	for _, source := range config.NodeSources {
		if strings.TrimSpace(source.Name) == "" {
			return errors.New("node_sources[].name 不能为空")
		}
		if _, exists := sourceNames[source.Name]; exists {
			return fmt.Errorf("重复的节点来源名称: %s", source.Name)
		}
		sourceNames[source.Name] = source
		if err := validateNodeSource(source); err != nil {
			return err
		}
	}

	routeNames := map[string]RouteConfig{}
	for _, route := range config.Routes {
		if strings.TrimSpace(route.Name) == "" {
			return errors.New("routes[].name 不能为空")
		}
		if _, exists := routeNames[route.Name]; exists {
			return fmt.Errorf("重复的线路名称: %s", route.Name)
		}
		if route.Entry.Source == "" {
			return fmt.Errorf("线路 %s 缺少节点来源", route.Name)
		}
		if _, exists := sourceNames[route.Entry.Source]; !exists {
			return fmt.Errorf("线路 %s 绑定的节点来源 %s 不存在", route.Name, route.Entry.Source)
		}
		if route.Landing.Source != "" {
			if _, exists := sourceNames[route.Landing.Source]; !exists {
				return fmt.Errorf("线路 %s 绑定的落地来源 %s 不存在", route.Name, route.Landing.Source)
			}
		}
		routeNames[route.Name] = route
	}

	if config.Usage.SelectedRoute != "" {
		if _, exists := routeNames[config.Usage.SelectedRoute]; !exists {
			return fmt.Errorf("这台设备选择的线路 %s 不存在", config.Usage.SelectedRoute)
		}
	}
	if config.Usage.LocalProxy.Enabled {
		if config.Usage.LocalProxy.Port <= 0 {
			return errors.New("usage.local_proxy.port 必须大于 0")
		}
		if config.Usage.LocalProxy.Listen != "127.0.0.1" && len(config.Usage.LocalProxy.Users) == 0 {
			return errors.New("本机代理允许其他设备连接时必须设置账号和密码")
		}
	}

	publicationPorts := map[int]struct{}{}
	for _, publication := range config.Publications {
		if strings.TrimSpace(publication.Name) == "" {
			return errors.New("publications[].name 不能为空")
		}
		if _, exists := routeNames[publication.Route]; !exists {
			return fmt.Errorf("发布 %s 绑定的线路 %s 不存在", publication.Name, publication.Route)
		}
		if publication.Enabled && publication.Port <= 0 && publication.Type != "subscription" && publication.Type != "axis_connection" {
			return fmt.Errorf("发布 %s 缺少端口", publication.Name)
		}
		if publication.Enabled && publication.Auth.Username == "" && publication.Auth.Password == "" && publication.Auth.Token == "" {
			return fmt.Errorf("发布 %s 必须设置账号密码或令牌", publication.Name)
		}
		if publication.Port > 0 {
			if _, exists := publicationPorts[publication.Port]; exists {
				return fmt.Errorf("重复的发布端口: %d", publication.Port)
			}
			publicationPorts[publication.Port] = struct{}{}
		}
	}
	return nil
}

func validateNodeSource(source NodeSource) error {
	switch source.Type {
	case "subscription":
		if source.Subscription.URL == "" {
			return fmt.Errorf("节点来源 %s 缺少订阅链接", source.Name)
		}
	case "proxy":
		if source.Endpoint.Server == "" || source.Endpoint.Port <= 0 {
			return fmt.Errorf("节点来源 %s 缺少连接地址或端口", source.Name)
		}
	case "axis":
		if source.Axis.URL == "" {
			return fmt.Errorf("节点来源 %s 缺少 AXIS 连接地址", source.Name)
		}
	case "local_node":
		if source.LocalNode.Port <= 0 {
			return fmt.Errorf("本机节点 %s 缺少端口", source.Name)
		}
		if !isSupportedLocalNodeProtocol(source.Protocol) {
			return fmt.Errorf("本机节点 %s 的协议不支持: %s", source.Name, source.Protocol)
		}
		if source.LocalNode.Route == "" {
			return fmt.Errorf("本机节点 %s 缺少绑定线路", source.Name)
		}
		if source.LocalNode.ExternalHost == "" {
			return fmt.Errorf("本机节点 %s 缺少对外域名", source.Name)
		}
		if len(source.LocalNode.Users) == 0 {
			return fmt.Errorf("本机节点 %s 必须设置账号和密码", source.Name)
		}
	default:
		return fmt.Errorf("节点来源 %s 类型不支持: %s", source.Name, source.Type)
	}
	return nil
}

func isSupportedLocalNodeProtocol(protocol string) bool {
	switch protocol {
	case "http", "socks", "socks5", "trojan", "hysteria2":
		return true
	default:
		return false
	}
}

func NormalizeConfigForSave(config *Config, configPath string) *Config {
	out := mustJSONClone(*config)
	out.Admin.RequiresPasswordReset = false
	out.Subscriptions = nil
	out.LandingProxies = nil
	out.EgressGroups = nil
	out.TransitRoutes = nil
	out.Listeners = nil
	if out.Server.Host == "" {
		out.Server.Host = "127.0.0.1"
	}
	if out.Server.Port == 0 {
		out.Server.Port = 8787
	}
	if out.Admin.Username == "" {
		out.Admin.Username = "admin"
	}
	if out.Admin.SessionTTLHours == 0 {
		out.Admin.SessionTTLHours = 12
	}
	if out.Runtime.Workdir == "" {
		out.Runtime.Workdir = ResolveDefaultRuntimeWorkdir(configPath)
	}
	if out.Runtime.MihomoBinary == "" {
		out.Runtime.MihomoBinary = "mihomo"
	}
	if out.Runtime.ExternalController == "" {
		out.Runtime.ExternalController = "http://127.0.0.1:9090"
	}
	deriveLegacyRuntimeConfig(&out)
	return &out
}

func WriteConfig(configPath string, config *Config) (*Config, error) {
	normalized := NormalizeConfigForSave(config, configPath)
	if err := ValidateConfig(normalized); err != nil {
		return nil, err
	}
	forSave := mustJSONClone(*normalized)
	forSave.Subscriptions = nil
	forSave.LandingProxies = nil
	forSave.EgressGroups = nil
	forSave.TransitRoutes = nil
	forSave.Listeners = nil
	body, err := yaml.Marshal(forSave)
	if err != nil {
		return nil, err
	}
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, body, 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tempPath, configPath); err != nil {
		return nil, err
	}
	return normalized, nil
}

func deriveLegacyRuntimeConfig(config *Config) {
	config.Subscriptions = buildLegacySubscriptions(config.NodeSources)
	config.LandingProxies = buildLegacyLandings(config.NodeSources)
	config.EgressGroups = buildLegacyGroups(config.Routes)
	config.TransitRoutes = nil
	config.Listeners = buildLegacyListeners(config)
}

func buildLegacySubscriptions(sources []NodeSource) []Subscription {
	items := []Subscription{}
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		switch source.Type {
		case "subscription", "axis":
			url := source.Subscription.URL
			if source.Type == "axis" {
				url = source.Axis.URL
			}
			items = append(items, Subscription{Name: source.Name, Type: firstNonEmpty(source.Protocol, "mihomo-http"), URL: url, Interval: source.Subscription.Interval, Enabled: true, HealthCheckURL: source.Subscription.HealthCheckURL, HealthCheckInterval: source.Subscription.HealthCheckInterval, Headers: source.Subscription.Headers, Via: source.Subscription.Via})
		case "proxy", "local_node":
			endpoint := source.Endpoint
			if source.Type == "local_node" {
				endpoint = NodeSourceEndpoint{Server: firstNonEmpty(source.LocalNode.Listen, "127.0.0.1"), Port: source.LocalNode.Port, Username: firstLocalUser(source.LocalNode.Users).Username, Password: firstLocalUser(source.LocalNode.Users).Password, TLS: source.LocalNode.TLS, SNI: source.LocalNode.SNI}
			}
			items = append(items, Subscription{Name: source.Name, Type: firstNonEmpty(source.Protocol, "socks5"), Server: endpoint.Server, Port: endpoint.Port, Username: endpoint.Username, Password: endpoint.Password, TLS: endpoint.TLS, SNI: endpoint.SNI, SkipCertVerify: endpoint.SkipCertVerify, Enabled: true, Interval: 3600, HealthCheckURL: "https://www.gstatic.com/generate_204", HealthCheckInterval: 300})
		}
	}
	return items
}

func buildLegacyLandings(sources []NodeSource) []LandingProxy {
	items := []LandingProxy{}
	for _, source := range sources {
		if !source.Enabled || (source.Type != "proxy" && source.Type != "local_node") {
			continue
		}
		endpoint := source.Endpoint
		if source.Type == "local_node" {
			endpoint = NodeSourceEndpoint{Server: firstNonEmpty(source.LocalNode.Listen, "127.0.0.1"), Port: source.LocalNode.Port, Username: firstLocalUser(source.LocalNode.Users).Username, Password: firstLocalUser(source.LocalNode.Users).Password, TLS: source.LocalNode.TLS, SNI: source.LocalNode.SNI}
		}
		items = append(items, LandingProxy{Name: source.Name, Type: normalizeLandingProtocol(source.Protocol), Server: endpoint.Server, Port: endpoint.Port, Username: endpoint.Username, Password: endpoint.Password, TLS: endpoint.TLS, SNI: endpoint.SNI, SkipCertVerify: endpoint.SkipCertVerify, Enabled: true})
	}
	return items
}

func buildLegacyGroups(routes []RouteConfig) []EgressGroup {
	items := []EgressGroup{}
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		items = append(items, EgressGroup{Name: route.Name, Provider: route.Entry.Source, Mode: normalizeRouteStrategy(route.Strategy), Filter: route.Entry.Node, LandingProxy: route.Landing.Source, HealthCheckURL: route.HealthCheck.URL, Interval: route.HealthCheck.Interval})
	}
	return items
}

func buildLegacyListeners(config *Config) []Listener {
	items := []Listener{}
	if config.Usage.LocalProxy.Enabled && config.Usage.SelectedRoute != "" {
		items = append(items, Listener{Name: "本机代理", Type: firstNonEmpty(config.Usage.LocalProxy.Type, "mixed"), Listen: firstNonEmpty(config.Usage.LocalProxy.Listen, "127.0.0.1"), Port: config.Usage.LocalProxy.Port, UDP: true, Enabled: true, Users: config.Usage.LocalProxy.Users, RouteMode: "direct", EgressGroup: config.Usage.SelectedRoute})
	}
	for _, publication := range config.Publications {
		if !publication.Enabled || publication.Type != "http_proxy" {
			continue
		}
		items = append(items, Listener{Name: publication.Name, Type: "http", Listen: firstNonEmpty(publication.Listen, "0.0.0.0"), Port: publication.Port, UDP: false, Enabled: true, Users: []ListenerUser{{Username: publication.Auth.Username, Password: publication.Auth.Password}}, RouteMode: "direct", EgressGroup: publication.Route})
	}
	for _, source := range config.NodeSources {
		if !source.Enabled || source.Type != "local_node" {
			continue
		}
		items = append(items, Listener{Name: source.Name, Type: normalizeLocalNodeListenerType(source.Protocol), Listen: firstNonEmpty(source.LocalNode.Listen, "127.0.0.1"), Port: source.LocalNode.Port, UDP: source.Protocol == "hysteria2", Enabled: true, Users: source.LocalNode.Users, Certificate: source.LocalNode.Certificate, PrivateKey: source.LocalNode.PrivateKey, SNI: source.LocalNode.SNI, RouteMode: "direct", EgressGroup: source.LocalNode.Route})
	}
	return items
}

func normalizeLocalNodeListenerType(protocol string) string {
	switch protocol {
	case "socks5":
		return "socks"
	default:
		return protocol
	}
}

func firstLocalUser(users []ListenerUser) ListenerUser {
	if len(users) == 0 {
		return ListenerUser{}
	}
	return users[0]
}

func normalizeLandingProtocol(protocol string) string {
	switch protocol {
	case "http", "trojan", "hysteria2":
		return protocol
	default:
		return "socks5"
	}
}

func normalizeRouteStrategy(strategy string) string {
	switch strategy {
	case "auto":
		return "url-test"
	case "fallback":
		return "fallback"
	default:
		return "manual"
	}
}

func EnsureRuntimeLayout(config *Config, configPath string) (*RuntimeLayout, error) {
	rootDir := filepath.Dir(configPath)
	runtimeDir := ResolveRuntimeDir(configPath, config.Runtime.Workdir)
	providersDir := filepath.Join(runtimeDir, "providers")
	versionsDir := filepath.Join(runtimeDir, "mihomo", "versions")
	if err := os.MkdirAll(providersDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(versionsDir, 0o755); err != nil {
		return nil, err
	}

	return &RuntimeLayout{
		RootDir:            rootDir,
		RuntimeDir:         runtimeDir,
		ProvidersDir:       providersDir,
		MihomoConfigPath:   filepath.Join(runtimeDir, "mihomo.yaml"),
		LastGoodConfigPath: filepath.Join(runtimeDir, "mihomo.last-good.yaml"),
		StatePath:          filepath.Join(runtimeDir, "control-state.json"),
		DatabasePath:       filepath.Join(runtimeDir, "axis.db"),
		MihomoVersionsDir:  versionsDir,
	}, nil
}
