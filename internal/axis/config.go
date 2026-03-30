package axis

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
		Subscriptions: parsed.Subscriptions,
		EgressGroups:  parsed.EgressGroups,
		Listeners:     parsed.Listeners,
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
		if subscription.URL == "" {
			return fmt.Errorf("订阅 %s 缺少 url", subscription.Name)
		}
		if _, exists := providerNames[subscription.Name]; exists {
			return fmt.Errorf("重复的订阅名称: %s", subscription.Name)
		}
		providerNames[subscription.Name] = struct{}{}
	}

	groupNames := map[string]struct{}{}
	for _, group := range config.EgressGroups {
		if group.Name == "" {
			return errors.New("egress_groups[].name 不能为空")
		}
		if group.Provider == "" {
			return fmt.Errorf("出口组 %s 缺少订阅源", group.Name)
		}
		if _, err := compilePattern(group.Filter); err != nil {
			return err
		}
		if _, err := compilePattern(group.ExcludeFilter); err != nil {
			return err
		}
		if _, exists := groupNames[group.Name]; exists {
			return fmt.Errorf("重复的出口组名称: %s", group.Name)
		}
		groupNames[group.Name] = struct{}{}
	}

	ports := map[int]struct{}{}
	for _, listener := range config.Listeners {
		if listener.Name == "" {
			return errors.New("listeners[].name 不能为空")
		}
		if listener.Port <= 0 {
			return fmt.Errorf("监听 %s 缺少 port", listener.Name)
		}
		if listener.EgressGroup == "" {
			return fmt.Errorf("监听 %s 缺少 egress_group", listener.Name)
		}
		if _, exists := ports[listener.Port]; exists {
			return fmt.Errorf("重复的监听端口: %d", listener.Port)
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
