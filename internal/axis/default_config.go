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
			Password:              "admin",
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
		Subscriptions:  []Subscription{},
		LandingProxies: []LandingProxy{},
		EgressGroups:   []EgressGroup{},
		TransitRoutes:  []TransitRoute{},
		Listeners:      []Listener{},
	}
}
