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

func LoadConfig(configPath string) (*Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var parsed Config
	if len(raw) > 0 {
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
	}

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
		Subscriptions:  parsed.Subscriptions,
		LandingProxies: parsed.LandingProxies,
		EgressGroups:   parsed.EgressGroups,
		TransitRoutes:  parsed.TransitRoutes,
		Listeners:      parsed.Listeners,
	}

	if parsed.Runtime.Workdir != "" || parsed.Runtime.MihomoBinary != "" || parsed.Runtime.ExternalController != "" || parsed.Runtime.ExternalSecret != "" || parsed.Runtime.RenderOnly {
		config.Runtime.RenderOnly = parsed.Runtime.RenderOnly
	}

	if config.Admin.Password == "" && config.Admin.PasswordHash == "" {
		config.Admin.Password = "admin"
	}

	for index := range config.Subscriptions {
		if config.Subscriptions[index].Type == "" {
			config.Subscriptions[index].Type = "mihomo-http"
		}
		if config.Subscriptions[index].Interval == 0 {
			config.Subscriptions[index].Interval = 3600
		}
		if config.Subscriptions[index].HealthCheckURL == "" {
			config.Subscriptions[index].HealthCheckURL = "https://www.gstatic.com/generate_204"
		}
		if config.Subscriptions[index].HealthCheckInterval == 0 {
			config.Subscriptions[index].HealthCheckInterval = 300
		}
		if !config.Subscriptions[index].Enabled {
			config.Subscriptions[index].Enabled = parsed.Subscriptions[index].Enabled
		}
		if !parsed.Subscriptions[index].Enabled {
			config.Subscriptions[index].Enabled = false
		} else if !config.Subscriptions[index].Enabled {
			config.Subscriptions[index].Enabled = true
		}
	}

	for index := range config.EgressGroups {
		if config.EgressGroups[index].Mode == "" {
			config.EgressGroups[index].Mode = "manual"
		}
		if config.EgressGroups[index].Mode == "fallback" {
			config.EgressGroups[index].Proxies = normalizeProxyOrder(config.EgressGroups[index].Proxies)
			if config.EgressGroups[index].HealthCheckURL == "" {
				config.EgressGroups[index].HealthCheckURL = "https://www.gstatic.com/generate_204"
			}
			if config.EgressGroups[index].Interval == 0 {
				config.EgressGroups[index].Interval = 60
			}
		}
	}

	for index := range config.LandingProxies {
		if config.LandingProxies[index].Type == "" {
			config.LandingProxies[index].Type = "socks5"
		}
		if !parsed.LandingProxies[index].Enabled {
			config.LandingProxies[index].Enabled = false
		} else if !config.LandingProxies[index].Enabled {
			config.LandingProxies[index].Enabled = true
		}
	}

	for index := range config.TransitRoutes {
		if !parsed.TransitRoutes[index].Enabled {
			config.TransitRoutes[index].Enabled = false
		} else if !config.TransitRoutes[index].Enabled {
			config.TransitRoutes[index].Enabled = true
		}
	}

	for index := range config.Listeners {
		if config.Listeners[index].Type == "" {
			config.Listeners[index].Type = "socks"
		}
		if config.Listeners[index].Listen == "" {
			config.Listeners[index].Listen = "0.0.0.0"
		}
		if !parsed.Listeners[index].Enabled {
			config.Listeners[index].Enabled = false
		} else if !config.Listeners[index].Enabled {
			config.Listeners[index].Enabled = true
		}
		if !parsed.Listeners[index].UDP {
			config.Listeners[index].UDP = false
		} else if !config.Listeners[index].UDP {
			config.Listeners[index].UDP = true
		}
		config.Listeners[index].RouteMode = normalizeListenerRouteMode(config.Listeners[index].RouteMode, config.Listeners[index].TransitRoute)
	}

	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

func ValidateConfig(config *Config) error {
	if config.Server.Port <= 0 {
		return errors.New("server.port 必须大于 0")
	}
	if config.Admin.Username == "" {
		return errors.New("admin.username 不能为空")
	}

	providerNames := map[string]struct{}{}
	for _, subscription := range config.Subscriptions {
		if subscription.Name == "" {
			return errors.New("subscriptions[].name 不能为空")
		}
		if isManualProviderType(subscription.Type) {
			if strings.TrimSpace(subscription.Server) == "" {
				return fmt.Errorf("单节点 %s 缺少 server", subscription.Name)
			}
			if subscription.Port <= 0 {
				return fmt.Errorf("单节点 %s 缺少 port", subscription.Name)
			}
		} else if subscription.URL == "" {
			return fmt.Errorf("订阅 %s 缺少 url", subscription.Name)
		}
		if _, exists := providerNames[subscription.Name]; exists {
			return fmt.Errorf("重复的订阅名称: %s", subscription.Name)
		}
		providerNames[subscription.Name] = struct{}{}
	}

	landingNames := map[string]struct{}{}
	for _, landing := range config.LandingProxies {
		if landing.Name == "" {
			return errors.New("landing_proxies[].name 不能为空")
		}
		if landing.Server == "" {
			return fmt.Errorf("落地节点 %s 缺少 server", landing.Name)
		}
		if landing.Port <= 0 {
			return fmt.Errorf("落地节点 %s 缺少 port", landing.Name)
		}
		switch landing.Type {
		case "http", "socks5":
		default:
			return fmt.Errorf("落地节点 %s 暂不支持类型 %s", landing.Name, landing.Type)
		}
		if _, exists := landingNames[landing.Name]; exists {
			return fmt.Errorf("重复的落地节点名称: %s", landing.Name)
		}
		landingNames[landing.Name] = struct{}{}
	}

	groupNames := map[string]struct{}{}
	for _, group := range config.EgressGroups {
		if group.Name == "" {
			return errors.New("egress_groups[].name 不能为空")
		}
		if group.Provider == "" {
			return fmt.Errorf("出口线路 %s 缺少订阅源", group.Name)
		}
		if _, err := compilePattern(group.Filter); err != nil {
			return err
		}
		if _, err := compilePattern(group.ExcludeFilter); err != nil {
			return err
		}
		if group.Mode == "fallback" && len(normalizeProxyOrder(group.Proxies)) == 0 {
			return fmt.Errorf("顺序容灾组 %s 至少需要选择一个节点", group.Name)
		}
		if _, exists := groupNames[group.Name]; exists {
			return fmt.Errorf("重复的出口线路名称: %s", group.Name)
		}
		if group.LandingProxy != "" {
			if _, exists := landingNames[group.LandingProxy]; !exists {
				return fmt.Errorf("出口线路 %s 绑定的落地节点 %s 不存在", group.Name, group.LandingProxy)
			}
		}
		groupNames[group.Name] = struct{}{}
	}

	transitRouteNames := map[string]struct{}{}
	for _, route := range config.TransitRoutes {
		if route.Name == "" {
			return errors.New("transit_routes[].name 不能为空")
		}
		if strings.TrimSpace(route.UpstreamProvider) == "" {
			return fmt.Errorf("中转线路 %s 缺少 upstream_provider", route.Name)
		}
		if strings.TrimSpace(route.UpstreamProxyName) == "" {
			return fmt.Errorf("中转线路 %s 缺少 upstream_proxy_name", route.Name)
		}
		if strings.TrimSpace(route.EgressGroup) == "" {
			return fmt.Errorf("中转线路 %s 缺少 egress_group", route.Name)
		}
		if _, exists := providerNames[route.UpstreamProvider]; !exists {
			return fmt.Errorf("中转线路 %s 绑定的来源 %s 不存在", route.Name, route.UpstreamProvider)
		}
		if _, exists := groupNames[route.EgressGroup]; !exists {
			return fmt.Errorf("中转线路 %s 绑定的出口线路 %s 不存在", route.Name, route.EgressGroup)
		}
		if _, exists := transitRouteNames[route.Name]; exists {
			return fmt.Errorf("重复的中转线路名称: %s", route.Name)
		}
		transitRouteNames[route.Name] = struct{}{}
	}

	ports := map[int]struct{}{}
	for _, listener := range config.Listeners {
		if listener.Name == "" {
			return errors.New("listeners[].name 不能为空")
		}
		if listener.Port <= 0 {
			return fmt.Errorf("本地入口 %s 缺少 port", listener.Name)
		}
		switch getListenerRouteMode(listener) {
		case "transit":
			if strings.TrimSpace(listener.TransitRoute) == "" {
				return fmt.Errorf("本地入口 %s 缺少 transit_route", listener.Name)
			}
			if _, exists := transitRouteNames[listener.TransitRoute]; !exists {
				return fmt.Errorf("本地入口 %s 绑定的中转线路 %s 不存在", listener.Name, listener.TransitRoute)
			}
		default:
			if listener.EgressGroup == "" {
				return fmt.Errorf("本地入口 %s 缺少 egress_group", listener.Name)
			}
			if _, exists := groupNames[listener.EgressGroup]; !exists {
				return fmt.Errorf("本地入口 %s 绑定的出口线路 %s 不存在", listener.Name, listener.EgressGroup)
			}
		}
		if _, exists := ports[listener.Port]; exists {
			return fmt.Errorf("重复的本地入口端口: %d", listener.Port)
		}
		ports[listener.Port] = struct{}{}
	}

	return nil
}

func NormalizeConfigForSave(config *Config, configPath string) *Config {
	out := mustJSONClone(*config)
	out.Admin.RequiresPasswordReset = false

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

	return &out
}

func WriteConfig(configPath string, config *Config) (*Config, error) {
	normalized := NormalizeConfigForSave(config, configPath)
	if err := ValidateConfig(normalized); err != nil {
		return nil, err
	}

	body, err := yaml.Marshal(normalized)
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

func EnsureRuntimeLayout(config *Config, configPath string) (*RuntimeLayout, error) {
	rootDir := filepath.Dir(configPath)
	runtimeDir := filepath.Clean(filepath.Join(rootDir, config.Runtime.Workdir))
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
