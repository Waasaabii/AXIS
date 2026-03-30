package axis

type ServerConfig struct {
	Host           string `json:"host" yaml:"host"`
	Port           int    `json:"port" yaml:"port"`
	StartupRefresh bool   `json:"startup_refresh" yaml:"startup_refresh"`
}

type AdminConfig struct {
	Username              string `json:"username" yaml:"username"`
	Password              string `json:"password" yaml:"password"`
	PasswordHash          string `json:"password_hash" yaml:"password_hash"`
	RequiresPasswordReset bool   `json:"requires_password_reset,omitempty" yaml:"-"`
	SessionSecret         string `json:"session_secret" yaml:"session_secret"`
	SessionTTLHours       int    `json:"session_ttl_hours" yaml:"session_ttl_hours"`
}

type RuntimeConfig struct {
	Workdir            string `json:"workdir" yaml:"workdir"`
	MihomoBinary       string `json:"mihomo_binary" yaml:"mihomo_binary"`
	ExternalController string `json:"external_controller" yaml:"external_controller"`
	ExternalSecret     string `json:"external_secret" yaml:"external_secret"`
	RenderOnly         bool   `json:"render_only" yaml:"render_only"`
}

type Subscription struct {
	Name                string            `json:"name" yaml:"name"`
	Type                string            `json:"type" yaml:"type"`
	URL                 string            `json:"url" yaml:"url"`
	Interval            int               `json:"interval" yaml:"interval"`
	Enabled             bool              `json:"enabled" yaml:"enabled"`
	HealthCheckURL      string            `json:"health_check_url,omitempty" yaml:"health_check_url,omitempty"`
	HealthCheckInterval int               `json:"health_check_interval,omitempty" yaml:"health_check_interval,omitempty"`
	Headers             map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Via                 string            `json:"via,omitempty" yaml:"via,omitempty"`
}

type LandingProxy struct {
	Name           string `json:"name" yaml:"name"`
	Type           string `json:"type" yaml:"type"`
	Server         string `json:"server" yaml:"server"`
	Port           int    `json:"port" yaml:"port"`
	Username       string `json:"username,omitempty" yaml:"username,omitempty"`
	Password       string `json:"password,omitempty" yaml:"password,omitempty"`
	TLS            bool   `json:"tls,omitempty" yaml:"tls,omitempty"`
	SNI            string `json:"sni,omitempty" yaml:"sni,omitempty"`
	SkipCertVerify bool   `json:"skip_cert_verify,omitempty" yaml:"skip_cert_verify,omitempty"`
	Enabled        bool   `json:"enabled" yaml:"enabled"`
}

type EgressGroup struct {
	Name           string `json:"name" yaml:"name"`
	Provider       string `json:"provider" yaml:"provider"`
	Mode           string `json:"mode" yaml:"mode"`
	Filter         string `json:"filter" yaml:"filter"`
	ExcludeFilter  string `json:"exclude_filter" yaml:"exclude_filter"`
	LandingProxy   string `json:"landing_proxy,omitempty" yaml:"landing_proxy,omitempty"`
	Strategy       string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	HealthCheckURL string `json:"health_check_url,omitempty" yaml:"health_check_url,omitempty"`
	Interval       int    `json:"interval,omitempty" yaml:"interval,omitempty"`
	Fallback       string `json:"fallback,omitempty" yaml:"fallback,omitempty"`
}

type ListenerUser struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

type Listener struct {
	Name        string         `json:"name" yaml:"name"`
	Type        string         `json:"type" yaml:"type"`
	Listen      string         `json:"listen" yaml:"listen"`
	Port        int            `json:"port" yaml:"port"`
	UDP         bool           `json:"udp" yaml:"udp"`
	Enabled     bool           `json:"enabled" yaml:"enabled"`
	Users       []ListenerUser `json:"users" yaml:"users"`
	EgressGroup string         `json:"egress_group" yaml:"egress_group"`
}

type Config struct {
	Server         ServerConfig   `json:"server" yaml:"server"`
	Admin          AdminConfig    `json:"admin" yaml:"admin"`
	Runtime        RuntimeConfig  `json:"runtime" yaml:"runtime"`
	Subscriptions  []Subscription `json:"subscriptions" yaml:"subscriptions"`
	LandingProxies []LandingProxy `json:"landing_proxies" yaml:"landing_proxies"`
	EgressGroups   []EgressGroup  `json:"egress_groups" yaml:"egress_groups"`
	Listeners      []Listener     `json:"listeners" yaml:"listeners"`
}

type RuntimeLayout struct {
	RootDir            string `json:"rootDir"`
	RuntimeDir         string `json:"runtimeDir"`
	ProvidersDir       string `json:"providersDir"`
	MihomoConfigPath   string `json:"mihomoConfigPath"`
	LastGoodConfigPath string `json:"lastGoodConfigPath"`
	StatePath          string `json:"statePath"`
	DatabasePath       string `json:"databasePath"`
	MihomoVersionsDir  string `json:"mihomoVersionsDir"`
}

type RuntimeState struct {
	Mode                string `json:"mode"`
	MihomoBinary        string `json:"mihomoBinary,omitempty"`
	MihomoBinaryFound   bool   `json:"mihomoBinaryFound"`
	Controller          string `json:"controller,omitempty"`
	ControllerReachable bool   `json:"controllerReachable"`
	ConfigPath          string `json:"configPath,omitempty"`
	LastRenderAt        string `json:"lastRenderAt,omitempty"`
	LastApplyAt         string `json:"lastApplyAt,omitempty"`
	LastApplyStatus     string `json:"lastApplyStatus,omitempty"`
	LastApplyMessage    string `json:"lastApplyMessage,omitempty"`
}

type ControllerState struct {
	Reachable bool   `json:"reachable"`
	Mode      string `json:"mode"`
	Version   string `json:"version,omitempty"`
	Message   string `json:"message"`
	CheckedAt string `json:"checkedAt,omitempty"`
}

type NodeInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Server  string `json:"server"`
	Port    int    `json:"port"`
	Network string `json:"network,omitempty"`
	TLS     string `json:"tls,omitempty"`
	Source  string `json:"source,omitempty"`
}

type ProviderRecord struct {
	Provider    string     `json:"provider"`
	URLMasked   string     `json:"urlMasked,omitempty"`
	RefreshedAt string     `json:"refreshedAt,omitempty"`
	NodeCount   int        `json:"nodeCount"`
	Nodes       []NodeInfo `json:"nodes"`
	LastError   string     `json:"lastError,omitempty"`
}

type GroupState struct {
	LastHealthcheckAt string `json:"lastHealthcheckAt,omitempty"`
}

type EventEntry struct {
	ID      string `json:"id"`
	Level   string `json:"level"`
	Scope   string `json:"scope"`
	Message string `json:"message"`
	At      string `json:"at"`
}

type AppState struct {
	StartedAt       string                    `json:"startedAt"`
	Runtime         RuntimeState              `json:"runtime"`
	Controller      *ControllerState          `json:"controller,omitempty"`
	Providers       map[string]ProviderRecord `json:"providers"`
	GroupSelections map[string]string         `json:"groupSelections"`
	Groups          map[string]GroupState     `json:"groups,omitempty"`
	Events          []EventEntry              `json:"events"`
}

type GroupCandidate struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	NodeName string `json:"nodeName,omitempty"`
}

type GroupView struct {
	Name              string           `json:"name"`
	Mode              string           `json:"mode"`
	Provider          string           `json:"provider"`
	Filter            string           `json:"filter"`
	CandidateCount    int              `json:"candidateCount"`
	Current           string           `json:"current,omitempty"`
	CurrentValue      string           `json:"currentValue,omitempty"`
	Candidates        []GroupCandidate `json:"candidates"`
	LastHealthcheckAt string           `json:"lastHealthcheckAt,omitempty"`
	LandingProxy      string           `json:"landingProxy,omitempty"`
	RouteSummary      string           `json:"routeSummary,omitempty"`
	ProviderMissing   bool             `json:"providerMissing"`
	ProviderDisabled  bool             `json:"providerDisabled"`
	LandingMissing    bool             `json:"landingMissing"`
	LandingDisabled   bool             `json:"landingDisabled"`
}

type ListenerView struct {
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Listen           string         `json:"listen"`
	Port             int            `json:"port"`
	UDP              bool           `json:"udp"`
	Users            []ListenerUser `json:"users"`
	UserCount        int            `json:"userCount"`
	EgressGroup      string         `json:"egressGroup"`
	CurrentProxy     string         `json:"currentProxy,omitempty"`
	RouteSummary     string         `json:"routeSummary,omitempty"`
	Status           string         `json:"status"`
	GroupMissing     bool           `json:"groupMissing"`
	ProviderMissing  bool           `json:"providerMissing"`
	ProviderDisabled bool           `json:"providerDisabled"`
	LandingMissing   bool           `json:"landingMissing"`
	LandingDisabled  bool           `json:"landingDisabled"`
}

type LandingProxyView struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	Server         string   `json:"server"`
	Port           int      `json:"port"`
	Username       string   `json:"username,omitempty"`
	Password       string   `json:"password,omitempty"`
	TLS            bool     `json:"tls"`
	SNI            string   `json:"sni,omitempty"`
	SkipCertVerify bool     `json:"skipCertVerify"`
	Enabled        bool     `json:"enabled"`
	InUseBy        []string `json:"inUseBy"`
	RouteCount     int      `json:"routeCount"`
}

type RuntimeCheck struct {
	Key            string `json:"key"`
	Title          string `json:"title"`
	OK             bool   `json:"ok"`
	Level          string `json:"level"`
	Summary        string `json:"summary"`
	Recommendation string `json:"recommendation,omitempty"`
}

type RuntimePreflightSummary struct {
	Ready    bool `json:"ready"`
	Total    int  `json:"total"`
	Passed   int  `json:"passed"`
	Warnings int  `json:"warnings"`
	Errors   int  `json:"errors"`
	Info     int  `json:"info"`
}

type RuntimePreflightPaths struct {
	ConfigPath            string `json:"configPath"`
	RuntimeDir            string `json:"runtimeDir"`
	ProvidersDir          string `json:"providersDir"`
	MihomoConfigPath      string `json:"mihomoConfigPath"`
	LastGoodConfigPath    string `json:"lastGoodConfigPath"`
	StatePath             string `json:"statePath"`
	ProxyrelayServicePath string `json:"proxyrelayServicePath"`
	MihomoServicePath     string `json:"mihomoServicePath"`
}

type RuntimePreflightSnapshot struct {
	GeneratedAt     string                  `json:"generatedAt"`
	Mode            string                  `json:"mode"`
	Status          string                  `json:"status"`
	Ready           bool                    `json:"ready"`
	Summary         RuntimePreflightSummary `json:"summary"`
	Paths           RuntimePreflightPaths   `json:"paths"`
	Checks          []RuntimeCheck          `json:"checks"`
	Recommendations []string                `json:"recommendations"`
}

type SetupCheck struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Ready   bool   `json:"ready"`
	Summary string `json:"summary"`
	Action  string `json:"action,omitempty"`
}

type SetupState struct {
	Required             bool         `json:"required"`
	NeedsPasswordReset   bool         `json:"needsPasswordReset"`
	HasSubscriptions     bool         `json:"hasSubscriptions"`
	HasRealSubscriptions bool         `json:"hasRealSubscriptions"`
	HasEgressGroups      bool         `json:"hasEgressGroups"`
	HasListeners         bool         `json:"hasListeners"`
	Checks               []SetupCheck `json:"checks"`
	Reasons              []string     `json:"reasons"`
}

type MihomoVersionSupport struct {
	Version         string `json:"version"`
	Recommended     bool   `json:"recommended"`
	Supported       bool   `json:"supported"`
	Compatibility   string `json:"compatibility"`
	ReleaseURL      string `json:"releaseUrl"`
	AssetURL        string `json:"assetUrl,omitempty"`
	AssetName       string `json:"assetName,omitempty"`
	CurrentPlatform bool   `json:"currentPlatform"`
}

type MihomoVersionRecord struct {
	Version      string `json:"version"`
	Status       string `json:"status"`
	ArchivePath  string `json:"archivePath,omitempty"`
	BinaryPath   string `json:"binaryPath,omitempty"`
	DownloadedAt string `json:"downloadedAt,omitempty"`
	InstalledAt  string `json:"installedAt,omitempty"`
	ActivatedAt  string `json:"activatedAt,omitempty"`
	LastError    string `json:"lastError,omitempty"`
}

type MihomoVersionsResponse struct {
	Platform          map[string]string      `json:"platform"`
	Recommended       string                 `json:"recommended"`
	ActiveVersion     string                 `json:"activeVersion,omitempty"`
	ConfiguredBinary  string                 `json:"configuredBinary"`
	SupportMatrix     []MihomoVersionSupport `json:"supportMatrix"`
	InstalledVersions []MihomoVersionRecord  `json:"installedVersions"`
}
