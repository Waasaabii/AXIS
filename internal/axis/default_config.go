package axis

func BuildDefaultConfig(configPath string) *Config {
	return &Config{
		Server: ServerConfig{
			Host:           "127.0.0.1",
			Port:           8787,
			StartupRefresh: false,
		},
		Admin: AdminConfig{
			Username:              "admin",
			Password:              "",
			PasswordHash:          "",
			SessionSecret:         "",
			SessionTTLHours:       12,
			RequiresPasswordReset: true,
		},
		Runtime: RuntimeConfig{
			Workdir:            ResolveDefaultRuntimeWorkdir(configPath),
			MihomoBinary:       "mihomo",
			ExternalController: "http://127.0.0.1:9090",
			ExternalSecret:     "",
			RenderOnly:         true,
		},
		NodeSources:  []NodeSource{},
		Routes:       []RouteConfig{},
		Usage:        UsageConfig{LocalProxy: LocalProxyUsage{Type: "mixed", Listen: "127.0.0.1", Port: 7890}, VirtualInterface: VirtualInterfaceUsage{Mode: "system"}},
		Publications: []PublicationConfig{},
		LLM:          LLMConfig{Endpoint: "responses", TimeoutSeconds: 60},
	}
}
