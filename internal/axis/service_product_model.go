package axis

import (
	"fmt"
	"strings"
)

type UsageView struct {
	SelectedRoute    string                `json:"selectedRoute,omitempty"`
	LocalProxy       LocalProxyUsage       `json:"localProxy"`
	VirtualInterface VirtualInterfaceUsage `json:"virtualInterface"`
	CurrentRoute     *RouteConfig          `json:"currentRoute,omitempty"`
	Ready            bool                  `json:"ready"`
	MissingSteps     []string              `json:"missingSteps"`
	LastAppliedAt    string                `json:"lastAppliedAt,omitempty"`
	LastApplyStatus  string                `json:"lastApplyStatus,omitempty"`
	LastApplyMessage string                `json:"lastApplyMessage,omitempty"`
}

type RuntimeOverview struct {
	App          map[string]any `json:"app"`
	Runtime      RuntimeState   `json:"runtime"`
	Controller   any            `json:"controller,omitempty"`
	Counts       map[string]int `json:"counts"`
	Warnings     []string       `json:"warnings"`
	RecentEvents []EventEntry   `json:"recentEvents"`
}

func (s *Service) GetNodeSources() []NodeSource {
	return mustJSONClone(s.config.NodeSources)
}

func (s *Service) GetRoutes() []RouteConfig {
	return mustJSONClone(s.config.Routes)
}

func (s *Service) GetUsage() UsageView {
	usage := s.config.Usage
	missing := []string{}
	if len(s.config.NodeSources) == 0 {
		missing = append(missing, "先添加节点来源。")
	}
	if len(s.config.Routes) == 0 {
		missing = append(missing, "再创建一条线路。")
	}
	if usage.SelectedRoute == "" {
		missing = append(missing, "选择这台设备要使用的线路。")
	}
	if !usage.LocalProxy.Enabled && !usage.VirtualInterface.Enabled {
		missing = append(missing, "开启本机代理或虚拟网口。")
	}
	var current *RouteConfig
	for index := range s.config.Routes {
		if s.config.Routes[index].Name == usage.SelectedRoute {
			current = &s.config.Routes[index]
			break
		}
	}
	return UsageView{
		SelectedRoute:    usage.SelectedRoute,
		LocalProxy:       usage.LocalProxy,
		VirtualInterface: usage.VirtualInterface,
		CurrentRoute:     current,
		Ready:            len(missing) == 0,
		MissingSteps:     missing,
		LastAppliedAt:    usage.LastAppliedAt,
		LastApplyStatus:  usage.LastApplyStatus,
		LastApplyMessage: usage.LastApplyMessage,
	}
}

func (s *Service) GetPublications() []PublicationConfig {
	return mustJSONClone(s.config.Publications)
}

func (s *Service) saveProductConfig(next *Config) (map[string]any, int) {
	deriveLegacyRuntimeConfig(next)
	return s.SaveConfig(next)
}

func (s *Service) AddNodeSource(payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
	source := parseNodeSourcePayload(payload, NodeSource{Enabled: true})
	if source.Name == "" {
		return map[string]any{"ok": false, "error": "节点来源名称不能为空"}, 400
	}
	for _, item := range next.NodeSources {
		if item.Name == source.Name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("节点来源 %s 已存在", source.Name)}, 400
		}
	}
	next.NodeSources = append(next.NodeSources, source)
	return s.saveProductConfig(&next)
}

func (s *Service) UpdateNodeSource(name string, payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
	for index := range next.NodeSources {
		if next.NodeSources[index].Name == name {
			next.NodeSources[index] = parseNodeSourcePayload(payload, next.NodeSources[index])
			return s.saveProductConfig(&next)
		}
	}
	return map[string]any{"ok": false, "error": fmt.Sprintf("节点来源 %s 不存在", name)}, 404
}

func (s *Service) RemoveNodeSource(name string) (map[string]any, int) {
	next := s.cloneConfig()
	for _, route := range next.Routes {
		if route.Entry.Source == name || route.Landing.Source == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("节点来源 %s 正在被线路使用", name)}, 400
		}
	}
	filtered := []NodeSource{}
	found := false
	for _, item := range next.NodeSources {
		if item.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("节点来源 %s 不存在", name)}, 404
	}
	next.NodeSources = filtered
	return s.saveProductConfig(&next)
}

func (s *Service) AddRoute(payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
	route := parseRoutePayload(payload, RouteConfig{Enabled: true, Strategy: "manual"})
	if route.Name == "" {
		return map[string]any{"ok": false, "error": "线路名称不能为空"}, 400
	}
	for _, item := range next.Routes {
		if item.Name == route.Name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("线路 %s 已存在", route.Name)}, 400
		}
	}
	next.Routes = append(next.Routes, route)
	return s.saveProductConfig(&next)
}

func (s *Service) UpdateRoute(name string, payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
	for index := range next.Routes {
		if next.Routes[index].Name == name {
			next.Routes[index] = parseRoutePayload(payload, next.Routes[index])
			return s.saveProductConfig(&next)
		}
	}
	return map[string]any{"ok": false, "error": fmt.Sprintf("线路 %s 不存在", name)}, 404
}

func (s *Service) RemoveRoute(name string) (map[string]any, int) {
	next := s.cloneConfig()
	if next.Usage.SelectedRoute == name {
		return map[string]any{"ok": false, "error": fmt.Sprintf("线路 %s 正在被这台设备使用", name)}, 400
	}
	for _, publication := range next.Publications {
		if publication.Route == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("线路 %s 正在被发布项使用", name)}, 400
		}
	}
	filtered := []RouteConfig{}
	found := false
	for _, item := range next.Routes {
		if item.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("线路 %s 不存在", name)}, 404
	}
	next.Routes = filtered
	return s.saveProductConfig(&next)
}

func (s *Service) UpdateUsage(payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
	if payload["selectedRoute"] != nil || payload["selected_route"] != nil {
		next.Usage.SelectedRoute = firstNonEmpty(stringValue(payload["selectedRoute"]), stringValue(payload["selected_route"]))
	}
	if value, ok := payload["localProxy"].(map[string]any); ok {
		next.Usage.LocalProxy = parseLocalProxyUsage(value, next.Usage.LocalProxy)
	}
	if value, ok := payload["local_proxy"].(map[string]any); ok {
		next.Usage.LocalProxy = parseLocalProxyUsage(value, next.Usage.LocalProxy)
	}
	if value, ok := payload["virtualInterface"].(map[string]any); ok {
		next.Usage.VirtualInterface = parseVirtualInterfaceUsage(value, next.Usage.VirtualInterface)
	}
	if value, ok := payload["virtual_interface"].(map[string]any); ok {
		next.Usage.VirtualInterface = parseVirtualInterfaceUsage(value, next.Usage.VirtualInterface)
	}
	next.Usage.LastAppliedAt = nowISO()
	next.Usage.LastApplyStatus = "saved"
	next.Usage.LastApplyMessage = "当前使用设置已保存。"
	return s.saveProductConfig(&next)
}

func (s *Service) AddPublication(payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
	publication := parsePublicationPayload(payload, PublicationConfig{Enabled: true, AccessScope: "lan", Format: "url"})
	if publication.Name == "" {
		return map[string]any{"ok": false, "error": "发布名称不能为空"}, 400
	}
	for _, item := range next.Publications {
		if item.Name == publication.Name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("发布 %s 已存在", publication.Name)}, 400
		}
	}
	next.Publications = append(next.Publications, publication)
	return s.saveProductConfig(&next)
}

func (s *Service) UpdatePublication(name string, payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
	for index := range next.Publications {
		if next.Publications[index].Name == name {
			next.Publications[index] = parsePublicationPayload(payload, next.Publications[index])
			return s.saveProductConfig(&next)
		}
	}
	return map[string]any{"ok": false, "error": fmt.Sprintf("发布 %s 不存在", name)}, 404
}

func (s *Service) RemovePublication(name string) (map[string]any, int) {
	next := s.cloneConfig()
	filtered := []PublicationConfig{}
	found := false
	for _, item := range next.Publications {
		if item.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("发布 %s 不存在", name)}, 404
	}
	next.Publications = filtered
	return s.saveProductConfig(&next)
}

func parseNodeSourcePayload(payload map[string]any, current NodeSource) NodeSource {
	if payload["name"] != nil {
		current.Name = strings.TrimSpace(stringValue(payload["name"]))
	}
	if payload["type"] != nil {
		current.Type = strings.TrimSpace(stringValue(payload["type"]))
	}
	if payload["enabled"] != nil {
		current.Enabled = boolValue(payload["enabled"], true)
	}
	if payload["protocol"] != nil {
		current.Protocol = strings.TrimSpace(stringValue(payload["protocol"]))
	}
	if payload["notes"] != nil {
		current.Notes = stringValue(payload["notes"])
	}
	if value, ok := payload["subscription"].(map[string]any); ok {
		current.Subscription.URL = firstNonEmpty(stringValue(value["url"]), current.Subscription.URL)
		current.Subscription.Interval = intValue(value["interval"], current.Subscription.Interval)
		current.Subscription.HealthCheckURL = firstNonEmpty(stringValue(value["healthCheckUrl"]), stringValue(value["health_check_url"]), current.Subscription.HealthCheckURL)
		current.Subscription.HealthCheckInterval = intValue(firstAny(value["healthCheckInterval"], value["health_check_interval"]), current.Subscription.HealthCheckInterval)
	}
	if value, ok := payload["endpoint"].(map[string]any); ok {
		current.Endpoint.Server = firstNonEmpty(stringValue(value["server"]), current.Endpoint.Server)
		current.Endpoint.Port = intValue(value["port"], current.Endpoint.Port)
		current.Endpoint.Username = firstNonEmpty(stringValue(value["username"]), current.Endpoint.Username)
		current.Endpoint.Password = firstNonEmpty(stringValue(value["password"]), current.Endpoint.Password)
		current.Endpoint.TLS = boolValue(value["tls"], current.Endpoint.TLS)
		current.Endpoint.SNI = firstNonEmpty(stringValue(value["sni"]), current.Endpoint.SNI)
		current.Endpoint.SkipCertVerify = boolValue(firstAny(value["skipCertVerify"], value["skip_cert_verify"]), current.Endpoint.SkipCertVerify)
	}
	if value, ok := payload["axis"].(map[string]any); ok {
		current.Axis.URL = firstNonEmpty(stringValue(value["url"]), current.Axis.URL)
		current.Axis.Username = firstNonEmpty(stringValue(value["username"]), current.Axis.Username)
		current.Axis.Password = firstNonEmpty(stringValue(value["password"]), current.Axis.Password)
		current.Axis.Token = firstNonEmpty(stringValue(value["token"]), current.Axis.Token)
	}
	if value, ok := firstAny(payload["localNode"], payload["local_node"]).(map[string]any); ok {
		current.LocalNode.AccessMode = firstNonEmpty(stringValue(value["accessMode"]), stringValue(value["access_mode"]), current.LocalNode.AccessMode)
		current.LocalNode.Listen = firstNonEmpty(stringValue(value["listen"]), current.LocalNode.Listen)
		current.LocalNode.Port = intValue(value["port"], current.LocalNode.Port)
		current.LocalNode.ExternalHost = firstNonEmpty(stringValue(value["externalHost"]), stringValue(value["external_host"]), current.LocalNode.ExternalHost)
		current.LocalNode.ExternalPort = intValue(firstAny(value["externalPort"], value["external_port"]), current.LocalNode.ExternalPort)
		current.LocalNode.Route = firstNonEmpty(stringValue(value["route"]), current.LocalNode.Route)
		current.LocalNode.Certificate = firstNonEmpty(stringValue(value["certificate"]), current.LocalNode.Certificate)
		current.LocalNode.PrivateKey = firstNonEmpty(stringValue(value["privateKey"]), stringValue(value["private_key"]), current.LocalNode.PrivateKey)
		current.LocalNode.TLS = boolValue(value["tls"], current.LocalNode.TLS)
		current.LocalNode.SNI = firstNonEmpty(stringValue(value["sni"]), current.LocalNode.SNI)
		current.LocalNode.SkipCertVerify = boolValue(firstAny(value["skipCertVerify"], value["skip_cert_verify"]), current.LocalNode.SkipCertVerify)
		if value["users"] != nil {
			current.LocalNode.Users = parseUsers(value["users"])
		}
	}
	if current.Type == "" {
		current.Type = "subscription"
	}
	if current.Protocol == "" {
		current.Protocol = defaultNodeSourceProtocol(current)
	}
	return current
}

func parseRoutePayload(payload map[string]any, current RouteConfig) RouteConfig {
	if payload["name"] != nil {
		current.Name = strings.TrimSpace(stringValue(payload["name"]))
	}
	if payload["enabled"] != nil {
		current.Enabled = boolValue(payload["enabled"], true)
	}
	if payload["strategy"] != nil {
		current.Strategy = strings.TrimSpace(stringValue(payload["strategy"]))
	}
	if payload["notes"] != nil {
		current.Notes = stringValue(payload["notes"])
	}
	if value, ok := payload["entry"].(map[string]any); ok {
		current.Entry.Source = firstNonEmpty(stringValue(value["source"]), current.Entry.Source)
		current.Entry.Node = firstNonEmpty(stringValue(value["node"]), current.Entry.Node)
	}
	if value, ok := payload["landing"].(map[string]any); ok {
		current.Landing.Source = stringValue(value["source"])
		current.Landing.Node = stringValue(value["node"])
	}
	if value, ok := firstAny(payload["healthCheck"], payload["health_check"]).(map[string]any); ok {
		current.HealthCheck.URL = firstNonEmpty(stringValue(value["url"]), current.HealthCheck.URL)
		current.HealthCheck.Interval = intValue(value["interval"], current.HealthCheck.Interval)
	}
	return current
}

func parseLocalProxyUsage(payload map[string]any, current LocalProxyUsage) LocalProxyUsage {
	if payload["enabled"] != nil {
		current.Enabled = boolValue(payload["enabled"], current.Enabled)
	}
	current.Type = firstNonEmpty(stringValue(payload["type"]), current.Type)
	current.Listen = firstNonEmpty(stringValue(payload["listen"]), current.Listen)
	current.Port = intValue(payload["port"], current.Port)
	if payload["users"] != nil {
		current.Users = parseUsers(payload["users"])
	}
	return current
}

func parseVirtualInterfaceUsage(payload map[string]any, current VirtualInterfaceUsage) VirtualInterfaceUsage {
	if payload["enabled"] != nil {
		current.Enabled = boolValue(payload["enabled"], current.Enabled)
	}
	current.Mode = firstNonEmpty(stringValue(payload["mode"]), current.Mode)
	current.Status = firstNonEmpty(stringValue(payload["status"]), current.Status)
	current.Message = firstNonEmpty(stringValue(payload["message"]), current.Message)
	return current
}

func parsePublicationPayload(payload map[string]any, current PublicationConfig) PublicationConfig {
	if payload["name"] != nil {
		current.Name = strings.TrimSpace(stringValue(payload["name"]))
	}
	if payload["type"] != nil {
		current.Type = strings.TrimSpace(stringValue(payload["type"]))
	}
	if payload["enabled"] != nil {
		current.Enabled = boolValue(payload["enabled"], true)
	}
	current.Route = firstNonEmpty(stringValue(payload["route"]), current.Route)
	current.Listen = firstNonEmpty(stringValue(payload["listen"]), current.Listen)
	current.Port = intValue(payload["port"], current.Port)
	current.AccessScope = firstNonEmpty(stringValue(payload["accessScope"]), stringValue(payload["access_scope"]), current.AccessScope)
	current.Format = firstNonEmpty(stringValue(payload["format"]), current.Format)
	if value, ok := payload["auth"].(map[string]any); ok {
		current.Auth.Username = firstNonEmpty(stringValue(value["username"]), current.Auth.Username)
		current.Auth.Password = firstNonEmpty(stringValue(value["password"]), current.Auth.Password)
		current.Auth.Token = firstNonEmpty(stringValue(value["token"]), current.Auth.Token)
	}
	return current
}

func firstAny(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
