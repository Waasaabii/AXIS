package axis

type ServerConfig struct {
	Host           string `json:"host" yaml:"host"`
	Port           int    `json:"port" yaml:"port"`
	StartupRefresh bool   `json:"startup_refresh" yaml:"startup_refresh"`
}

type AdminConfig struct {
	Username              string `json:"username" yaml:"username"`
	Password              string `json:"password,omitempty" yaml:"password"`
	PasswordHash          string `json:"password_hash,omitempty" yaml:"password_hash"`
	RequiresPasswordReset bool   `json:"requires_password_reset,omitempty" yaml:"-"`
	SessionSecret         string `json:"session_secret,omitempty" yaml:"session_secret"`
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
	URL                 string            `json:"url,omitempty" yaml:"url,omitempty"`
	Server              string            `json:"server,omitempty" yaml:"server,omitempty"`
	Port                int               `json:"port,omitempty" yaml:"port,omitempty"`
	Username            string            `json:"username,omitempty" yaml:"username,omitempty"`
	Password            string            `json:"password,omitempty" yaml:"password,omitempty"`
	TLS                 bool              `json:"tls,omitempty" yaml:"tls,omitempty"`
	SNI                 string            `json:"sni,omitempty" yaml:"sni,omitempty"`
	SkipCertVerify      bool              `json:"skip_cert_verify,omitempty" yaml:"skip_cert_verify,omitempty"`
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
	Name           string   `json:"name" yaml:"name"`
	Provider       string   `json:"provider" yaml:"provider"`
	Mode           string   `json:"mode" yaml:"mode"`
	Filter         string   `json:"filter" yaml:"filter"`
	ExcludeFilter  string   `json:"exclude_filter" yaml:"exclude_filter"`
	Proxies        []string `json:"proxies,omitempty" yaml:"proxies,omitempty"`
	LandingProxy   string   `json:"landing_proxy,omitempty" yaml:"landing_proxy,omitempty"`
	Strategy       string   `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	HealthCheckURL string   `json:"health_check_url,omitempty" yaml:"health_check_url,omitempty"`
	Interval       int      `json:"interval,omitempty" yaml:"interval,omitempty"`
	Fallback       string   `json:"fallback,omitempty" yaml:"fallback,omitempty"`
}

type TransitRoute struct {
	Name              string `json:"name" yaml:"name"`
	Enabled           bool   `json:"enabled" yaml:"enabled"`
	UpstreamProvider  string `json:"upstream_provider" yaml:"upstream_provider"`
	UpstreamProxyName string `json:"upstream_proxy_name" yaml:"upstream_proxy_name"`
	EgressGroup       string `json:"egress_group" yaml:"egress_group"`
	Notes             string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type ListenerUser struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

type Listener struct {
	Name         string         `json:"name" yaml:"name"`
	Type         string         `json:"type" yaml:"type"`
	Listen       string         `json:"listen" yaml:"listen"`
	Port         int            `json:"port" yaml:"port"`
	UDP          bool           `json:"udp" yaml:"udp"`
	Enabled      bool           `json:"enabled" yaml:"enabled"`
	Users        []ListenerUser `json:"users" yaml:"users"`
	Certificate  string         `json:"certificate,omitempty" yaml:"certificate,omitempty"`
	PrivateKey   string         `json:"private_key,omitempty" yaml:"private_key,omitempty"`
	SNI          string         `json:"sni,omitempty" yaml:"sni,omitempty"`
	RouteMode    string         `json:"route_mode,omitempty" yaml:"route_mode,omitempty"`
	EgressGroup  string         `json:"egress_group,omitempty" yaml:"egress_group,omitempty"`
	TransitRoute string         `json:"transit_route,omitempty" yaml:"transit_route,omitempty"`
}

type NodeSourceSubscription struct {
	URL                 string            `json:"url,omitempty" yaml:"url,omitempty"`
	Interval            int               `json:"interval,omitempty" yaml:"interval,omitempty"`
	HealthCheckURL      string            `json:"health_check_url,omitempty" yaml:"health_check_url,omitempty"`
	HealthCheckInterval int               `json:"health_check_interval,omitempty" yaml:"health_check_interval,omitempty"`
	Headers             map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Via                 string            `json:"via,omitempty" yaml:"via,omitempty"`
}

type NodeSourceEndpoint struct {
	Server         string `json:"server,omitempty" yaml:"server,omitempty"`
	Port           int    `json:"port,omitempty" yaml:"port,omitempty"`
	Username       string `json:"username,omitempty" yaml:"username,omitempty"`
	Password       string `json:"password,omitempty" yaml:"password,omitempty"`
	TLS            bool   `json:"tls,omitempty" yaml:"tls,omitempty"`
	SNI            string `json:"sni,omitempty" yaml:"sni,omitempty"`
	SkipCertVerify bool   `json:"skip_cert_verify,omitempty" yaml:"skip_cert_verify,omitempty"`
}

type AxisConnectionConfig struct {
	URL      string `json:"url,omitempty" yaml:"url,omitempty"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	Token    string `json:"token,omitempty" yaml:"token,omitempty"`
}

type LocalNodeConfig struct {
	AccessMode     string         `json:"access_mode,omitempty" yaml:"access_mode,omitempty"`
	Listen         string         `json:"listen,omitempty" yaml:"listen,omitempty"`
	Port           int            `json:"port,omitempty" yaml:"port,omitempty"`
	ExternalHost   string         `json:"external_host,omitempty" yaml:"external_host,omitempty"`
	ExternalPort   int            `json:"external_port,omitempty" yaml:"external_port,omitempty"`
	Users          []ListenerUser `json:"users,omitempty" yaml:"users,omitempty"`
	Route          string         `json:"route,omitempty" yaml:"route,omitempty"`
	Certificate    string         `json:"certificate,omitempty" yaml:"certificate,omitempty"`
	PrivateKey     string         `json:"private_key,omitempty" yaml:"private_key,omitempty"`
	TLS            bool           `json:"tls,omitempty" yaml:"tls,omitempty"`
	SNI            string         `json:"sni,omitempty" yaml:"sni,omitempty"`
	SkipCertVerify bool           `json:"skip_cert_verify,omitempty" yaml:"skip_cert_verify,omitempty"`
}

type NodeSource struct {
	Name         string                 `json:"name" yaml:"name"`
	Type         string                 `json:"type" yaml:"type"`
	Enabled      bool                   `json:"enabled" yaml:"enabled"`
	Protocol     string                 `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Subscription NodeSourceSubscription `json:"subscription,omitempty" yaml:"subscription,omitempty"`
	Endpoint     NodeSourceEndpoint     `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Axis         AxisConnectionConfig   `json:"axis,omitempty" yaml:"axis,omitempty"`
	LocalNode    LocalNodeConfig        `json:"local_node,omitempty" yaml:"local_node,omitempty"`
	Notes        string                 `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type RouteEndpointRef struct {
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
	Node   string `json:"node,omitempty" yaml:"node,omitempty"`
}

type RouteHealthCheck struct {
	URL      string `json:"url,omitempty" yaml:"url,omitempty"`
	Interval int    `json:"interval,omitempty" yaml:"interval,omitempty"`
}

type RouteConfig struct {
	Name        string           `json:"name" yaml:"name"`
	Enabled     bool             `json:"enabled" yaml:"enabled"`
	Entry       RouteEndpointRef `json:"entry,omitempty" yaml:"entry,omitempty"`
	Strategy    string           `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	Landing     RouteEndpointRef `json:"landing,omitempty" yaml:"landing,omitempty"`
	HealthCheck RouteHealthCheck `json:"health_check,omitempty" yaml:"health_check,omitempty"`
	Notes       string           `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type LocalProxyUsage struct {
	Enabled bool           `json:"enabled" yaml:"enabled"`
	Type    string         `json:"type" yaml:"type"`
	Listen  string         `json:"listen" yaml:"listen"`
	Port    int            `json:"port" yaml:"port"`
	Users   []ListenerUser `json:"users,omitempty" yaml:"users,omitempty"`
}

type VirtualInterfaceUsage struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Mode    string `json:"mode,omitempty" yaml:"mode,omitempty"`
	Status  string `json:"status,omitempty" yaml:"status,omitempty"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

type UsageConfig struct {
	SelectedRoute    string                `json:"selectedRoute,omitempty" yaml:"selected_route,omitempty"`
	LocalProxy       LocalProxyUsage       `json:"localProxy" yaml:"local_proxy"`
	VirtualInterface VirtualInterfaceUsage `json:"virtualInterface" yaml:"virtual_interface"`
	LastAppliedAt    string                `json:"lastAppliedAt,omitempty" yaml:"last_applied_at,omitempty"`
	LastApplyStatus  string                `json:"lastApplyStatus,omitempty" yaml:"last_apply_status,omitempty"`
	LastApplyMessage string                `json:"lastApplyMessage,omitempty" yaml:"last_apply_message,omitempty"`
}

type PublicationAuth struct {
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	Token    string `json:"token,omitempty" yaml:"token,omitempty"`
}

type PublicationConfig struct {
	Name        string          `json:"name" yaml:"name"`
	Type        string          `json:"type" yaml:"type"`
	Enabled     bool            `json:"enabled" yaml:"enabled"`
	Route       string          `json:"route" yaml:"route"`
	Listen      string          `json:"listen,omitempty" yaml:"listen,omitempty"`
	Port        int             `json:"port,omitempty" yaml:"port,omitempty"`
	Auth        PublicationAuth `json:"auth" yaml:"auth"`
	AccessScope string          `json:"accessScope,omitempty" yaml:"access_scope,omitempty"`
	Format      string          `json:"format,omitempty" yaml:"format,omitempty"`
}

type Config struct {
	Server       ServerConfig        `json:"server" yaml:"server"`
	Admin        AdminConfig         `json:"admin" yaml:"admin"`
	Runtime      RuntimeConfig       `json:"runtime" yaml:"runtime"`
	NodeSources  []NodeSource        `json:"node_sources" yaml:"node_sources"`
	Routes       []RouteConfig       `json:"routes" yaml:"routes"`
	Usage        UsageConfig         `json:"usage" yaml:"usage"`
	Publications []PublicationConfig `json:"publications" yaml:"publications"`

	Subscriptions  []Subscription `json:"-" yaml:"-"`
	LandingProxies []LandingProxy `json:"-" yaml:"-"`
	EgressGroups   []EgressGroup  `json:"-" yaml:"-"`
	TransitRoutes  []TransitRoute `json:"-" yaml:"-"`
	Listeners      []Listener     `json:"-" yaml:"-"`
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

type TransitRouteState struct {
	LastTestedAt    string `json:"lastTestedAt,omitempty"`
	LastTestStatus  string `json:"lastTestStatus,omitempty"`
	LastTestMessage string `json:"lastTestMessage,omitempty"`
	LastTestDelay   int    `json:"lastTestDelay,omitempty"`
	LastTestURL     string `json:"lastTestUrl,omitempty"`
}

type EventEntry struct {
	ID      string `json:"id"`
	Level   string `json:"level"`
	Scope   string `json:"scope"`
	Message string `json:"message"`
	At      string `json:"at"`
}

type HostState struct {
	AutostartEnabled bool   `json:"autostartEnabled"`
	DesktopMode      bool   `json:"desktopMode"`
	LastOpenedAt     string `json:"lastOpenedAt,omitempty"`
}

type AppState struct {
	StartedAt       string                       `json:"startedAt"`
	Runtime         RuntimeState                 `json:"runtime"`
	Controller      *ControllerState             `json:"controller,omitempty"`
	Providers       map[string]ProviderRecord    `json:"providers"`
	GroupSelections map[string]string            `json:"groupSelections"`
	Groups          map[string]GroupState        `json:"groups,omitempty"`
	TransitRoutes   map[string]TransitRouteState `json:"transitRoutes,omitempty"`
	Host            HostState                    `json:"host,omitempty"`
	Events          []EventEntry                 `json:"events"`
}

type HostStatus struct {
	Mode             string   `json:"mode"`
	DesktopMode      bool     `json:"desktopMode"`
	AutostartEnabled bool     `json:"autostartEnabled"`
	AutostartManaged bool     `json:"autostartManaged"`
	ConfigPath       string   `json:"configPath,omitempty"`
	RuntimeDir       string   `json:"runtimeDir,omitempty"`
	ListenAddress    string   `json:"listenAddress,omitempty"`
	Logs             []string `json:"logs,omitempty"`
}

type MainServiceStatus struct {
	Ready          bool   `json:"ready"`
	State          string `json:"state"`
	Mode           string `json:"mode"`
	Controller     string `json:"controller,omitempty"`
	ConfigPath     string `json:"configPath,omitempty"`
	BlockingReason string `json:"blockingReason,omitempty"`
	Message        string `json:"message"`
}

type UpdaterStatus struct {
	Mode            string `json:"mode"`
	Running         bool   `json:"running"`
	State           string `json:"state"`
	CurrentVersion  string `json:"currentVersion,omitempty"`
	LatestVersion   string `json:"latestVersion,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable"`
	CanAutoApply    bool   `json:"canAutoApply"`
	ReleaseURL      string `json:"releaseUrl,omitempty"`
	AssetName       string `json:"assetName,omitempty"`
	AssetURL        string `json:"assetUrl,omitempty"`
	CheckedAt       string `json:"checkedAt,omitempty"`
	Message         string `json:"message,omitempty"`
}

type BootstrapAuthStatus struct {
	Authenticated         bool   `json:"authenticated"`
	Username              string `json:"username,omitempty"`
	RequiresPasswordReset bool   `json:"requiresPasswordReset"`
}

type BootstrapStatus struct {
	Host           HostStatus          `json:"host"`
	MainService    MainServiceStatus   `json:"mainService"`
	Updater        UpdaterStatus       `json:"updater"`
	Setup          SetupState          `json:"setup"`
	Auth           BootstrapAuthStatus `json:"auth"`
	NextStep       string              `json:"nextStep"`
	BlockingReason string              `json:"blockingReason,omitempty"`
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
	Name                string           `json:"name"`
	Mode                string           `json:"mode"`
	Provider            string           `json:"provider"`
	Filter              string           `json:"filter"`
	ExcludeFilter       string           `json:"excludeFilter,omitempty"`
	CandidateCount      int              `json:"candidateCount"`
	Current             string           `json:"current,omitempty"`
	CurrentValue        string           `json:"currentValue,omitempty"`
	Candidates          []GroupCandidate `json:"candidates"`
	ProxyOrder          []string         `json:"proxyOrder,omitempty"`
	LastHealthcheckAt   string           `json:"lastHealthcheckAt,omitempty"`
	LandingProxy        string           `json:"landingProxy,omitempty"`
	HealthCheckURL      string           `json:"healthCheckURL,omitempty"`
	HealthCheckInterval int              `json:"healthCheckInterval,omitempty"`
	RouteSummary        string           `json:"routeSummary,omitempty"`
	ProviderMissing     bool             `json:"providerMissing"`
	ProviderDisabled    bool             `json:"providerDisabled"`
	LandingMissing      bool             `json:"landingMissing"`
	LandingDisabled     bool             `json:"landingDisabled"`
}

type ListenerView struct {
	Name                string         `json:"name"`
	Type                string         `json:"type"`
	Listen              string         `json:"listen"`
	Port                int            `json:"port"`
	UDP                 bool           `json:"udp"`
	Enabled             bool           `json:"enabled"`
	Users               []ListenerUser `json:"users"`
	UserCount           int            `json:"userCount"`
	RouteMode           string         `json:"routeMode"`
	EgressGroup         string         `json:"egressGroup"`
	TransitRoute        string         `json:"transitRoute,omitempty"`
	TargetName          string         `json:"targetName,omitempty"`
	CurrentProxy        string         `json:"currentProxy,omitempty"`
	RouteSummary        string         `json:"routeSummary,omitempty"`
	Status              string         `json:"status"`
	GroupMissing        bool           `json:"groupMissing"`
	ProviderMissing     bool           `json:"providerMissing"`
	ProviderDisabled    bool           `json:"providerDisabled"`
	LandingMissing      bool           `json:"landingMissing"`
	LandingDisabled     bool           `json:"landingDisabled"`
	TransitMissing      bool           `json:"transitMissing"`
	TransitDisabled     bool           `json:"transitDisabled"`
	TransitProxyMissing bool           `json:"transitProxyMissing"`
}

type TransitRouteView struct {
	Name                   string `json:"name"`
	Enabled                bool   `json:"enabled"`
	UpstreamProvider       string `json:"upstreamProvider"`
	UpstreamProxyName      string `json:"upstreamProxyName"`
	EgressGroup            string `json:"egressGroup"`
	Notes                  string `json:"notes,omitempty"`
	CurrentProxy           string `json:"currentProxy,omitempty"`
	CandidateCount         int    `json:"candidateCount"`
	EgressGroupMode        string `json:"egressGroupMode,omitempty"`
	RuntimeGroupName       string `json:"runtimeGroupName,omitempty"`
	RouteSummary           string `json:"routeSummary,omitempty"`
	LastTestedAt           string `json:"lastTestedAt,omitempty"`
	LastTestStatus         string `json:"lastTestStatus,omitempty"`
	LastTestMessage        string `json:"lastTestMessage,omitempty"`
	LastTestDelay          int    `json:"lastTestDelay,omitempty"`
	LastTestURL            string `json:"lastTestUrl,omitempty"`
	Status                 string `json:"status"`
	ProviderMissing        bool   `json:"providerMissing"`
	ProviderDisabled       bool   `json:"providerDisabled"`
	TransitProxyMissing    bool   `json:"transitProxyMissing"`
	EgressGroupMissing     bool   `json:"egressGroupMissing"`
	EgressProviderMissing  bool   `json:"egressProviderMissing"`
	EgressProviderDisabled bool   `json:"egressProviderDisabled"`
	LandingMissing         bool   `json:"landingMissing"`
	LandingDisabled        bool   `json:"landingDisabled"`
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
	AdminUsername        string       `json:"adminUsername"`
	HasNodeSources       bool         `json:"hasNodeSources"`
	HasRoutes            bool         `json:"hasRoutes"`
	HasUsage             bool         `json:"hasUsage"`
	HasPublications      bool         `json:"hasPublications"`
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
