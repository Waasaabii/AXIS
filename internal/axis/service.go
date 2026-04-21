package axis

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Service struct {
	configPath  string
	config      *Config
	layout      *RuntimeLayout
	state       *AppState
	store       *Store
	auth        *AuthContext
	controller  *MihomoControllerAdapter
	host        HostIntegration
	configMTime time.Time
}

func NewService(configPath string) (*Service, error) {
	if configPath == "" {
		resolvedPath, err := EnsureConfigPath("")
		if err != nil {
			return nil, err
		}
		configPath = resolvedPath
	}
	service := &Service{configPath: configPath}
	if err := service.LoadAll(); err != nil {
		return nil, err
	}
	return service, nil
}

func createEmptyState() *AppState {
	return &AppState{
		StartedAt:       nowISO(),
		Runtime:         RuntimeState{Mode: "bootstrap", LastApplyStatus: "idle"},
		Providers:       map[string]ProviderRecord{},
		GroupSelections: map[string]string{},
		Groups:          map[string]GroupState{},
		TransitRoutes:   map[string]TransitRouteState{},
		Host:            HostState{},
		Events:          []EventEntry{},
	}
}

func (s *Service) LoadAll() error {
	config, err := LoadConfig(s.configPath)
	if err != nil {
		return err
	}
	layout, err := EnsureRuntimeLayout(config, s.configPath)
	if err != nil {
		return err
	}
	store, err := NewStore(layout.DatabasePath)
	if err != nil {
		return err
	}
	if s.store != nil {
		_ = s.store.Close()
	}
	s.store = store
	s.config = config
	s.layout = layout
	s.auth = NewAuthContext(config.Admin.Username, config.Admin.PasswordHash, config.Admin.Password, config.Admin.SessionSecret, config.Admin.SessionTTLHours)
	s.controller = NewMihomoControllerAdapter(config.Runtime)
	s.host = NewNoopHostIntegration()

	if err := s.loadState(); err != nil {
		return err
	}
	s.detectRuntime()
	if _, err := s.renderRuntimeConfig(); err != nil {
		return err
	}
	s.detectController(false)
	if err := s.captureConfigModTime(); err != nil {
		return err
	}
	return s.persistState()
}

func (s *Service) loadState() error {
	state, err := s.store.LoadState()
	if err != nil {
		return err
	}
	if state != nil {
		s.state = normalizeState(state)
		s.state.Events, _ = s.store.ListEvents(100)
		return nil
	}

	if fileExists(s.layout.StatePath) {
		raw, err := os.ReadFile(s.layout.StatePath)
		if err == nil {
			var loaded AppState
			if json.Unmarshal(raw, &loaded) == nil {
				s.state = normalizeState(&loaded)
				s.state.Events, _ = s.store.ListEvents(100)
				return nil
			}
		}
	}

	s.state = normalizeState(createEmptyState())
	return nil
}

func (s *Service) persistState() error {
	s.state.Events, _ = s.store.ListEvents(100)
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.layout.StatePath, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return s.store.SaveState(s.state)
}

func (s *Service) pushEvent(level, scope, message string) {
	event := EventEntry{
		ID:      createNodeID(scope, nowISO()+message),
		Level:   level,
		Scope:   scope,
		Message: message,
		At:      nowISO(),
	}
	_ = s.store.AppendEvent(event, 100)
	s.state.Events, _ = s.store.ListEvents(100)
}

func (s *Service) detectRuntime() {
	binaryPath := pathOrResolved(s.configPath, s.config.Runtime.MihomoBinary)
	binaryFound := false
	if binaryPath != "" {
		if stat, err := os.Stat(binaryPath); err == nil && stat.Mode().IsRegular() {
			binaryFound = true
		}
	}
	s.state.Runtime = RuntimeState{
		Mode:                ternaryString(s.config.Runtime.RenderOnly, "render-only", "managed"),
		MihomoBinary:        firstNonEmpty(binaryPath, s.config.Runtime.MihomoBinary),
		MihomoBinaryFound:   binaryFound,
		Controller:          s.config.Runtime.ExternalController,
		ControllerReachable: false,
		ConfigPath:          s.layout.MihomoConfigPath,
		LastRenderAt:        s.state.Runtime.LastRenderAt,
		LastApplyAt:         s.state.Runtime.LastApplyAt,
		LastApplyStatus:     firstNonEmpty(s.state.Runtime.LastApplyStatus, "idle"),
		LastApplyMessage:    s.state.Runtime.LastApplyMessage,
	}
	if !binaryFound {
		s.pushEvent("warn", "runtime", fmt.Sprintf("未找到代理核心程序：%s", s.config.Runtime.MihomoBinary))
	}
}

func (s *Service) detectController(recordEvent bool) {
	probe := s.controller.Probe()
	probe.CheckedAt = nowISO()
	previous := s.state.Controller
	s.state.Controller = &probe
	s.state.Runtime.ControllerReachable = probe.Reachable
	if recordEvent && (previous == nil || previous.Reachable != probe.Reachable || previous.Message != probe.Message || previous.Version != probe.Version) {
		level := "warn"
		if probe.Reachable {
			level = "info"
		}
		s.pushEvent(level, "controller", probe.Message)
	}
}

func (s *Service) renderRuntimeConfig() (string, error) {
	rendered, err := RenderMihomoConfig(s.config)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(s.layout.MihomoConfigPath, []byte(rendered), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(s.layout.LastGoodConfigPath, []byte(rendered), 0o644); err != nil {
		return "", err
	}
	if err := s.writeManualProviderFiles(); err != nil {
		return "", err
	}
	s.state.Runtime.LastRenderAt = nowISO()
	s.pushEvent("info", "render", "最新代理核心配置已生成")
	return rendered, nil
}

func (s *Service) writeManualProviderFiles() error {
	for _, subscription := range s.config.Subscriptions {
		if !subscription.Enabled || !isManualProviderType(subscription.Type) {
			continue
		}
		content, err := buildManualProviderFileContent(subscription)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(s.layout.ProvidersDir, providerFileName(subscription.Name))
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) removeManualProviderFile(name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	targetPath := filepath.Join(s.layout.ProvidersDir, providerFileName(name))
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Service) Login(username, password string) (map[string]any, int) {
	if s.config.Admin.RequiresPasswordReset {
		return map[string]any{"ok": false, "error": "首次初始化还没完成，请先创建管理员账号和密码"}, 409
	}
	if !s.auth.VerifyCredentials(username, password) {
		s.pushEvent("warn", "auth", fmt.Sprintf("登录失败: %s", firstNonEmpty(username, "unknown")))
		_ = s.persistState()
		return map[string]any{"ok": false, "error": "账号或密码错误"}, 401
	}
	setupState := s.GetSetupState()
	s.pushEvent("info", "auth", fmt.Sprintf("登录成功: %s", username))
	_ = s.persistState()
	return map[string]any{
		"ok":                    true,
		"requiresPasswordReset": s.config.Admin.RequiresPasswordReset,
		"setupRequired":         setupState.Required,
		"user":                  map[string]string{"username": username},
	}, 200
}

func (s *Service) SessionStatus(valid bool, username string) map[string]any {
	if !valid {
		return map[string]any{"authenticated": false}
	}
	setupState := s.GetSetupState()
	return map[string]any{
		"authenticated":         true,
		"requiresPasswordReset": s.config.Admin.RequiresPasswordReset,
		"setupRequired":         setupState.Required,
		"user":                  map[string]string{"username": username},
	}
}

func isPlaceholderSubscriptionURL(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return true
	}
	return strings.Contains(trimmed, "example.com") || strings.Contains(trimmed, "your-subscription-url")
}

func hasUsableSubscriptionSource(subscription Subscription) bool {
	if isManualProviderType(subscription.Type) {
		return strings.TrimSpace(subscription.Server) != "" && subscription.Port > 0
	}
	return !isPlaceholderSubscriptionURL(subscription.URL)
}

func BuildSetupState(config *Config) SetupState {
	hasSubscriptions := len(config.Subscriptions) > 0
	hasRealSubscriptions := false
	for _, subscription := range config.Subscriptions {
		if hasUsableSubscriptionSource(subscription) {
			hasRealSubscriptions = true
			break
		}
	}

	hasEgressGroups := len(config.EgressGroups) > 0
	hasListeners := len(config.Listeners) > 0

	checks := []SetupCheck{
		{
			Key:     "password",
			Title:   "管理员密码",
			Ready:   !config.Admin.RequiresPasswordReset,
			Summary: ternaryString(config.Admin.RequiresPasswordReset, "请先设置管理员密码，避免他人直接进入控制台。", "管理员密码已设置。"),
			Action:  ternaryString(config.Admin.RequiresPasswordReset, "去设置密码", ""),
		},
		{
			Key:     "subscriptions",
			Title:   "订阅源",
			Ready:   hasRealSubscriptions,
			Summary: ternaryString(hasRealSubscriptions, "已添加可用订阅。", "还没填写可用的订阅链接，当前显示的是示例内容。"),
			Action:  ternaryString(hasRealSubscriptions, "", "添加订阅"),
		},
		{
			Key:     "egress-groups",
			Title:   "出口组",
			Ready:   hasEgressGroups,
			Summary: ternaryString(hasEgressGroups, "已创建代理分组。", "还没有分组，节点还不能按地区或用途整理。"),
			Action:  ternaryString(hasEgressGroups, "", "创建分组"),
		},
		{
			Key:     "listeners",
			Title:   "入口监听",
			Ready:   hasListeners,
			Summary: ternaryString(hasListeners, "已创建代理入口。", "还没有代理入口，其他设备暂时还不能连接。"),
			Action:  ternaryString(hasListeners, "", "创建入口"),
		},
	}

	reasons := []string{}
	if config.Admin.RequiresPasswordReset {
		reasons = append(reasons, "管理员密码还没设置。")
	}
	if !hasRealSubscriptions {
		reasons = append(reasons, "还没填写可用的订阅链接。")
	}
	if !hasEgressGroups {
		reasons = append(reasons, "还没有创建代理分组。")
	}
	if !hasListeners {
		reasons = append(reasons, "还没有创建代理入口。")
	}

	return SetupState{
		Required:             len(reasons) > 0,
		NeedsPasswordReset:   config.Admin.RequiresPasswordReset,
		AdminUsername:        config.Admin.Username,
		HasSubscriptions:     hasSubscriptions,
		HasRealSubscriptions: hasRealSubscriptions,
		HasEgressGroups:      hasEgressGroups,
		HasListeners:         hasListeners,
		Checks:               checks,
		Reasons:              reasons,
	}
}

func (s *Service) GetSetupState() SetupState {
	return BuildSetupState(s.config)
}

func (s *Service) UpdatePassword(password string) (map[string]any, int) {
	if password == "" {
		return map[string]any{"ok": false, "error": "新密码不能为空"}, 400
	}
	hash, err := CreatePasswordHash(password)
	if err != nil {
		return map[string]any{"ok": false, "error": "修改密码失败: " + err.Error()}, 500
	}
	s.config.Admin.PasswordHash = hash
	s.config.Admin.Password = ""
	s.config.Admin.RequiresPasswordReset = false
	if _, err := WriteConfig(s.configPath, s.config); err != nil {
		return map[string]any{"ok": false, "error": "修改密码失败: " + err.Error()}, 500
	}
	if err := s.LoadAll(); err != nil {
		return map[string]any{"ok": false, "error": "修改密码失败: " + err.Error()}, 500
	}
	return map[string]any{"ok": true}, 200
}

func (s *Service) BootstrapAdmin(username, password string) (map[string]any, int) {
	username = strings.TrimSpace(username)
	if !s.config.Admin.RequiresPasswordReset {
		return map[string]any{"ok": false, "error": "当前不是首次初始化状态，无需再创建管理员账号"}, 409
	}
	if username == "" {
		return map[string]any{"ok": false, "error": "管理员账号不能为空"}, 400
	}
	if password == "" {
		return map[string]any{"ok": false, "error": "管理员密码不能为空"}, 400
	}

	hash, err := CreatePasswordHash(password)
	if err != nil {
		return map[string]any{"ok": false, "error": "创建管理员账号失败: " + err.Error()}, 500
	}

	s.config.Admin.Username = username
	s.config.Admin.PasswordHash = hash
	s.config.Admin.Password = ""
	s.config.Admin.RequiresPasswordReset = false
	if _, err := WriteConfig(s.configPath, s.config); err != nil {
		return map[string]any{"ok": false, "error": "创建管理员账号失败: " + err.Error()}, 500
	}
	if err := s.LoadAll(); err != nil {
		return map[string]any{"ok": false, "error": "创建管理员账号失败: " + err.Error()}, 500
	}
	return map[string]any{"ok": true}, 200
}

func (s *Service) GetStatus() map[string]any {
	s.detectController(false)
	providers := s.GetProviders()
	groups := s.GetGroups()
	transitRoutes := s.GetTransitRoutes()
	listeners := s.GetListeners()
	return map[string]any{
		"app": map[string]any{
			"name":       "AXIS",
			"mode":       s.state.Runtime.Mode,
			"startedAt":  s.state.StartedAt,
			"renderOnly": s.config.Runtime.RenderOnly,
		},
		"runtime":    s.state.Runtime,
		"controller": s.state.Controller,
		"counts": map[string]int{
			"providers":     len(providers),
			"groups":        len(groups),
			"transitRoutes": len(transitRoutes),
			"listeners":     len(listeners),
			"nodes":         sumProviderNodes(providers),
		},
		"warnings":     s.buildWarnings(),
		"recentEvents": s.state.Events[:minInt(len(s.state.Events), 12)],
	}
}

func sumProviderNodes(providers []map[string]any) int {
	total := 0
	for _, item := range providers {
		if value, ok := item["nodeCount"].(int); ok {
			total += value
		}
	}
	return total
}

func (s *Service) buildWarnings() []string {
	warnings := []string{}
	if s.config.Admin.Password != "" {
		warnings = append(warnings, "管理员密码仍以明文方式写在配置里，建议改成加密保存。")
	}
	if !s.state.Runtime.MihomoBinaryFound {
		warnings = append(warnings, "还没找到代理核心程序，所以当前只能保存配置，不能直接接管运行。")
	}
	if !s.config.Runtime.RenderOnly && !s.state.Runtime.ControllerReachable {
		warnings = append(warnings, "当前已开启自动接管，但还连不上代理核心，请检查程序是否已启动。")
	}
	return warnings
}

func (s *Service) GetConfig() map[string]any {
	return map[string]any{
		"path":   s.configPath,
		"config": s.config,
	}
}

func (s *Service) SaveConfig(next *Config) (map[string]any, int) {
	if _, err := WriteConfig(s.configPath, next); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, 400
	}
	if err := s.LoadAll(); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	s.pushEvent("info", "config", "配置已保存并重新加载")
	runtimeApply := s.applyRuntimeConfig("配置保存")
	_ = s.persistState()
	return map[string]any{
		"ok":           true,
		"path":         s.configPath,
		"config":       s.config,
		"runtimeApply": runtimeApply,
	}, 200
}

func (s *Service) getProviderRecord(name string) ProviderRecord {
	if value, ok := s.state.Providers[name]; ok {
		return value
	}
	return ProviderRecord{Provider: name, Nodes: []NodeInfo{}}
}

func (s *Service) GetProviders() []map[string]any {
	items := make([]map[string]any, 0, len(s.config.Subscriptions))
	for _, subscription := range s.config.Subscriptions {
		record := s.getProviderRecord(subscription.Name)
		sourceKind := "subscription"
		endpoint := firstNonEmpty(record.URLMasked, subscription.URL)
		if isManualProviderType(subscription.Type) {
			sourceKind = "manual-node"
			endpoint = firstNonEmpty(record.URLMasked, buildManualProviderEndpoint(subscription))
		}
		items = append(items, map[string]any{
			"name":           subscription.Name,
			"type":           firstNonEmpty(subscription.Type, "mihomo-http"),
			"urlMasked":      endpoint,
			"endpoint":       endpoint,
			"interval":       max(subscription.Interval, 3600),
			"enabled":        subscription.Enabled,
			"refreshedAt":    record.RefreshedAt,
			"nodeCount":      record.NodeCount,
			"lastError":      record.LastError,
			"nodes":          record.Nodes,
			"sourceKind":     sourceKind,
			"manual":         isManualProviderType(subscription.Type),
			"hasCredentials": strings.TrimSpace(subscription.Username) != "" || subscription.Password != "",
		})
	}
	return items
}

func matchesGroup(group EgressGroup, node NodeInfo) bool {
	include := true
	if group.Filter != "" {
		regex, err := regexp.Compile(group.Filter)
		if err == nil {
			include = regex.MatchString(node.Name)
		}
	}
	exclude := false
	if group.ExcludeFilter != "" {
		regex, err := regexp.Compile(group.ExcludeFilter)
		if err == nil {
			exclude = regex.MatchString(node.Name)
		}
	}
	return include && !exclude
}

func relaySourceGroupName(groupName string) string {
	return groupName + "::source"
}

func landingProxyRuntimeName(name string) string {
	return "landing::" + name
}

func formatRouteSummary(groupName, currentProxy, landingProxy string) string {
	base := firstNonEmpty(currentProxy, groupName)
	if landingProxy == "" {
		return fmt.Sprintf("%s -> 公网", base)
	}
	return fmt.Sprintf("%s -> %s", base, landingProxy)
}

func formatTransitRouteSummary(upstreamProxy string, targetSummary string, egressGroup string) string {
	nextHop := firstNonEmpty(targetSummary, egressGroup)
	if nextHop == "" {
		nextHop = "公网"
	}
	if strings.TrimSpace(upstreamProxy) == "" {
		return nextHop
	}
	return fmt.Sprintf("%s -> %s", upstreamProxy, nextHop)
}

func buildTransitRouteBindingError(routeView *TransitRouteView) string {
	if routeView == nil {
		return "中转线路不存在"
	}
	if routeView.Status == "disabled" {
		return fmt.Sprintf("中转线路 %s 已停用", routeView.Name)
	}
	if routeView.ProviderMissing {
		return fmt.Sprintf("中转线路 %s 的来源已删除", routeView.Name)
	}
	if routeView.ProviderDisabled {
		return fmt.Sprintf("中转线路 %s 的来源已停用", routeView.Name)
	}
	if routeView.TransitProxyMissing {
		return fmt.Sprintf("中转线路 %s 当前不存在中转节点 %s", routeView.Name, routeView.UpstreamProxyName)
	}
	if routeView.EgressGroupMissing {
		return fmt.Sprintf("中转线路 %s 的落地出口组已删除", routeView.Name)
	}
	if routeView.EgressProviderMissing {
		return fmt.Sprintf("中转线路 %s 的落地出口 provider 已删除", routeView.Name)
	}
	if routeView.EgressProviderDisabled {
		return fmt.Sprintf("中转线路 %s 的落地出口 provider 已停用", routeView.Name)
	}
	if routeView.LandingMissing {
		return fmt.Sprintf("中转线路 %s 绑定的落地节点已删除", routeView.Name)
	}
	if routeView.LandingDisabled {
		return fmt.Sprintf("中转线路 %s 绑定的落地节点已停用", routeView.Name)
	}
	return fmt.Sprintf("中转线路 %s 当前不可用", routeView.Name)
}

func buildFallbackCandidates(group EgressGroup, nodes []NodeInfo) []GroupCandidate {
	nodeMap := map[string]NodeInfo{}
	for _, node := range nodes {
		nodeMap[node.Name] = node
	}

	candidates := make([]GroupCandidate, 0, len(group.Proxies))
	for _, proxyName := range normalizeProxyOrder(group.Proxies) {
		node, ok := nodeMap[proxyName]
		if !ok {
			continue
		}
		candidateID := node.Name
		candidateName := node.Name
		if group.LandingProxy != "" {
			candidateID = fmt.Sprintf("%s -> %s", node.Name, group.LandingProxy)
			candidateName = candidateID
		}
		candidates = append(candidates, GroupCandidate{
			ID:       candidateID,
			Name:     candidateName,
			Type:     node.Type,
			Server:   node.Server,
			Port:     node.Port,
			NodeName: node.Name,
		})
	}
	return candidates
}

func (s *Service) buildTransitRouteView(route TransitRoute, groupsByName map[string]GroupView, providerNames map[string]struct{}, enabledProviders map[string]struct{}) TransitRouteView {
	providerRecord := s.getProviderRecord(route.UpstreamProvider)
	transitProxyKnown := len(providerRecord.Nodes) > 0
	transitProxyMissing := false
	if transitProxyKnown {
		transitProxyMissing = true
		for _, node := range providerRecord.Nodes {
			if node.Name == route.UpstreamProxyName {
				transitProxyMissing = false
				break
			}
		}
	}

	targetGroup, egressGroupExists := groupsByName[route.EgressGroup]
	_, providerExists := providerNames[route.UpstreamProvider]
	_, providerEnabled := enabledProviders[route.UpstreamProvider]
	routeState := s.state.TransitRoutes[route.Name]

	status := "configured"
	if !route.Enabled {
		status = "disabled"
	} else if !providerExists || !egressGroupExists {
		status = "orphaned"
	} else if !providerEnabled || transitProxyMissing || targetGroup.ProviderMissing || targetGroup.ProviderDisabled || targetGroup.LandingMissing || targetGroup.LandingDisabled {
		status = "degraded"
	}

	return TransitRouteView{
		Name:                   route.Name,
		Enabled:                route.Enabled,
		UpstreamProvider:       route.UpstreamProvider,
		UpstreamProxyName:      route.UpstreamProxyName,
		EgressGroup:            route.EgressGroup,
		Notes:                  route.Notes,
		CurrentProxy:           targetGroup.Current,
		CandidateCount:         targetGroup.CandidateCount,
		EgressGroupMode:        targetGroup.Mode,
		RuntimeGroupName:       buildTransitMirrorGroupName(route.Name),
		RouteSummary:           formatTransitRouteSummary(route.UpstreamProxyName, targetGroup.RouteSummary, route.EgressGroup),
		LastTestedAt:           routeState.LastTestedAt,
		LastTestStatus:         routeState.LastTestStatus,
		LastTestMessage:        routeState.LastTestMessage,
		LastTestDelay:          routeState.LastTestDelay,
		LastTestURL:            routeState.LastTestURL,
		Status:                 status,
		ProviderMissing:        !providerExists,
		ProviderDisabled:       providerExists && !providerEnabled,
		TransitProxyMissing:    transitProxyMissing,
		EgressGroupMissing:     !egressGroupExists,
		EgressProviderMissing:  targetGroup.ProviderMissing,
		EgressProviderDisabled: targetGroup.ProviderDisabled,
		LandingMissing:         targetGroup.LandingMissing,
		LandingDisabled:        targetGroup.LandingDisabled,
	}
}

func (s *Service) buildGroupView(group EgressGroup) GroupView {
	providerRecord := s.getProviderRecord(group.Provider)
	candidates := []GroupCandidate{}
	if group.Mode == "fallback" {
		candidates = buildFallbackCandidates(group, providerRecord.Nodes)
	} else {
		for _, node := range providerRecord.Nodes {
			if matchesGroup(group, node) {
				candidateID := node.Name
				candidateName := node.Name
				if group.LandingProxy != "" {
					candidateID = fmt.Sprintf("%s -> %s", node.Name, group.LandingProxy)
					candidateName = candidateID
				}
				candidates = append(candidates, GroupCandidate{ID: candidateID, Name: candidateName, Type: node.Type, Server: node.Server, Port: node.Port, NodeName: node.Name})
			}
		}
	}
	currentValue := s.state.GroupSelections[group.Name]
	currentLabel := ""
	for _, candidate := range candidates {
		if candidate.ID == currentValue || (group.LandingProxy != "" && candidate.NodeName == currentValue) {
			currentValue = candidate.ID
			currentLabel = candidate.Name
			break
		}
	}
	if currentValue == "" && len(candidates) > 0 {
		currentValue = candidates[0].ID
		currentLabel = candidates[0].Name
	}
	if currentLabel == "" && len(candidates) > 0 {
		currentLabel = candidates[0].Name
	}
	groupState := s.state.Groups[group.Name]
	return GroupView{
		Name:                group.Name,
		Mode:                firstNonEmpty(group.Mode, "manual"),
		Provider:            group.Provider,
		Filter:              group.Filter,
		ExcludeFilter:       group.ExcludeFilter,
		CandidateCount:      len(candidates),
		Current:             currentLabel,
		CurrentValue:        currentValue,
		Candidates:          candidates,
		ProxyOrder:          normalizeProxyOrder(group.Proxies),
		LastHealthcheckAt:   groupState.LastHealthcheckAt,
		LandingProxy:        group.LandingProxy,
		HealthCheckURL:      group.HealthCheckURL,
		HealthCheckInterval: group.Interval,
		RouteSummary:        formatRouteSummary(group.Name, currentLabel, group.LandingProxy),
	}
}

func (s *Service) GetGroups() []GroupView {
	providerNames := map[string]struct{}{}
	enabledProviders := map[string]struct{}{}
	for _, subscription := range s.config.Subscriptions {
		providerNames[subscription.Name] = struct{}{}
		if subscription.Enabled {
			enabledProviders[subscription.Name] = struct{}{}
		}
	}
	landingNames := map[string]struct{}{}
	enabledLandings := map[string]struct{}{}
	for _, landing := range s.config.LandingProxies {
		landingNames[landing.Name] = struct{}{}
		if landing.Enabled {
			enabledLandings[landing.Name] = struct{}{}
		}
	}

	views := make([]GroupView, 0, len(s.config.EgressGroups))
	for _, group := range s.config.EgressGroups {
		view := s.buildGroupView(group)
		_, providerExists := providerNames[group.Provider]
		_, providerEnabled := enabledProviders[group.Provider]
		_, landingExists := landingNames[group.LandingProxy]
		_, landingEnabled := enabledLandings[group.LandingProxy]
		view.ProviderMissing = !providerExists
		view.ProviderDisabled = providerExists && !providerEnabled
		view.LandingMissing = group.LandingProxy != "" && !landingExists
		view.LandingDisabled = group.LandingProxy != "" && landingExists && !landingEnabled
		views = append(views, view)
	}
	return views
}

func (s *Service) GetTransitRoutes() []TransitRouteView {
	groupsByName := map[string]GroupView{}
	for _, group := range s.GetGroups() {
		groupsByName[group.Name] = group
	}

	providerNames := map[string]struct{}{}
	enabledProviders := map[string]struct{}{}
	for _, subscription := range s.config.Subscriptions {
		providerNames[subscription.Name] = struct{}{}
		if subscription.Enabled {
			enabledProviders[subscription.Name] = struct{}{}
		}
	}

	views := make([]TransitRouteView, 0, len(s.config.TransitRoutes))
	for _, route := range s.config.TransitRoutes {
		views = append(views, s.buildTransitRouteView(route, groupsByName, providerNames, enabledProviders))
	}
	return views
}

func (s *Service) getTransitRouteViewByName(name string) *TransitRouteView {
	for _, route := range s.GetTransitRoutes() {
		if route.Name == name {
			routeCopy := route
			return &routeCopy
		}
	}
	return nil
}

func (s *Service) GetListeners() []ListenerView {
	groupMap := map[string]GroupView{}
	for _, group := range s.GetGroups() {
		groupMap[group.Name] = group
	}
	transitRouteMap := map[string]TransitRouteView{}
	for _, route := range s.GetTransitRoutes() {
		transitRouteMap[route.Name] = route
	}
	groupNames := map[string]struct{}{}
	for _, group := range s.config.EgressGroups {
		groupNames[group.Name] = struct{}{}
	}
	transitRouteNames := map[string]struct{}{}
	for _, route := range s.config.TransitRoutes {
		transitRouteNames[route.Name] = struct{}{}
	}

	listeners := make([]ListenerView, 0, len(s.config.Listeners))
	for _, listener := range s.config.Listeners {
		routeMode := getListenerRouteMode(listener)
		group, groupOK := groupMap[listener.EgressGroup]
		_, groupExists := groupNames[listener.EgressGroup]
		transitRoute, transitOK := transitRouteMap[listener.TransitRoute]
		_, transitExists := transitRouteNames[listener.TransitRoute]

		status := "configured"
		groupMissing := false
		providerMissing := false
		providerDisabled := false
		landingMissing := false
		landingDisabled := false
		transitMissing := false
		transitDisabled := false
		transitProxyMissing := false
		currentProxy := ""
		routeSummary := ""
		egressGroup := listener.EgressGroup
		targetName := listener.EgressGroup

		if routeMode == "transit" {
			transitMissing = !transitExists
			transitDisabled = transitOK && transitRoute.Status == "disabled"
			transitProxyMissing = transitRoute.TransitProxyMissing
			providerMissing = transitRoute.EgressProviderMissing
			providerDisabled = transitRoute.EgressProviderDisabled
			landingMissing = transitRoute.LandingMissing
			landingDisabled = transitRoute.LandingDisabled
			currentProxy = transitRoute.CurrentProxy
			routeSummary = transitRoute.RouteSummary
			egressGroup = transitRoute.EgressGroup
			targetName = firstNonEmpty(transitRoute.Name, listener.TransitRoute)

			switch {
			case transitMissing:
				status = "orphaned"
			case transitDisabled:
				status = "disabled"
			case transitOK:
				status = transitRoute.Status
			default:
				status = "degraded"
			}
		} else {
			groupMissing = !groupExists
			providerMissing = group.ProviderMissing
			providerDisabled = group.ProviderDisabled
			landingMissing = group.LandingMissing
			landingDisabled = group.LandingDisabled
			if groupOK {
				currentProxy = group.Current
				routeSummary = group.RouteSummary
			}

			if groupMissing {
				status = "orphaned"
			} else if providerMissing || providerDisabled || landingMissing || landingDisabled {
				status = "degraded"
			}
		}

		listeners = append(listeners, ListenerView{
			Name:                listener.Name,
			Type:                firstNonEmpty(listener.Type, "socks"),
			Listen:              firstNonEmpty(listener.Listen, "0.0.0.0"),
			Port:                listener.Port,
			UDP:                 listener.UDP,
			Enabled:             listener.Enabled,
			Users:               listener.Users,
			UserCount:           len(listener.Users),
			RouteMode:           routeMode,
			EgressGroup:         egressGroup,
			TransitRoute:        listener.TransitRoute,
			TargetName:          targetName,
			CurrentProxy:        currentProxy,
			RouteSummary:        firstNonEmpty(routeSummary, targetName),
			Status:              status,
			GroupMissing:        groupMissing,
			ProviderMissing:     providerMissing,
			ProviderDisabled:    providerDisabled,
			LandingMissing:      landingMissing,
			LandingDisabled:     landingDisabled,
			TransitMissing:      transitMissing,
			TransitDisabled:     transitDisabled,
			TransitProxyMissing: transitProxyMissing,
		})
	}
	return listeners
}

func (s *Service) buildLandingProxyView(landing LandingProxy) LandingProxyView {
	inUseBy := []string{}
	for _, group := range s.config.EgressGroups {
		if group.LandingProxy == landing.Name {
			inUseBy = append(inUseBy, group.Name)
		}
	}
	return LandingProxyView{
		Name:           landing.Name,
		Type:           landing.Type,
		Server:         landing.Server,
		Port:           landing.Port,
		Username:       landing.Username,
		Password:       landing.Password,
		TLS:            landing.TLS,
		SNI:            landing.SNI,
		SkipCertVerify: landing.SkipCertVerify,
		Enabled:        landing.Enabled,
		InUseBy:        inUseBy,
		RouteCount:     len(inUseBy),
	}
}

func (s *Service) GetLandingProxies() []LandingProxyView {
	views := make([]LandingProxyView, 0, len(s.config.LandingProxies))
	for _, landing := range s.config.LandingProxies {
		views = append(views, s.buildLandingProxyView(landing))
	}
	return views
}

func (s *Service) GetEvents() []EventEntry {
	s.state.Events, _ = s.store.ListEvents(100)
	return s.state.Events
}

func (s *Service) GetRenderedConfig() (map[string]any, error) {
	content, err := os.ReadFile(s.layout.MihomoConfigPath)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"path":      s.layout.MihomoConfigPath,
		"updatedAt": s.state.Runtime.LastRenderAt,
		"content":   string(content),
	}, nil
}

func (s *Service) TestSubscription(payload map[string]any) (map[string]any, int) {
	urlValue, _ := payload["url"].(string)
	if urlValue == "" {
		return map[string]any{"ok": false, "error": "订阅 URL 不能为空"}, 400
	}
	name, _ := payload["name"].(string)
	subType, _ := payload["type"].(string)
	record, ok, _, _, err := FetchProviderSnapshot(Subscription{Name: firstNonEmpty(name, "test-provider"), Type: firstNonEmpty(subType, "mihomo-http"), URL: urlValue})
	if err != nil {
		s.pushEvent("error", "provider-test", "临时订阅测试失败")
		_ = s.persistState()
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	s.pushEvent(ternaryString(ok, "info", "warn"), "provider-test", fmt.Sprintf("临时订阅测试完成: %s", firstNonEmpty(name, "test-provider")))
	_ = s.persistState()
	return map[string]any{"ok": ok, "result": map[string]any{"previewNodes": record.Nodes}}, 200
}

func (s *Service) GetRuntimePreflight() RuntimePreflightSnapshot {
	s.detectController(false)
	return BuildRuntimePreflightSnapshot(s.configPath, s.config, s.layout, s.state, s.state.Controller)
}

func (s *Service) GetControllerStatus() map[string]any {
	s.detectController(false)
	if s.state.Controller == nil {
		return map[string]any{
			"baseUrl":          s.config.Runtime.ExternalController,
			"secretConfigured": s.config.Runtime.ExternalSecret != "",
			"renderOnly":       s.config.Runtime.RenderOnly,
			"reachable":        false,
			"mode":             ternaryString(s.config.Runtime.RenderOnly, "render-only", "managed"),
			"message":          "尚未检测",
		}
	}
	return map[string]any{
		"baseUrl":          s.config.Runtime.ExternalController,
		"secretConfigured": s.config.Runtime.ExternalSecret != "",
		"renderOnly":       s.config.Runtime.RenderOnly,
		"reachable":        s.state.Controller.Reachable,
		"mode":             s.state.Controller.Mode,
		"version":          s.state.Controller.Version,
		"message":          s.state.Controller.Message,
		"checkedAt":        s.state.Controller.CheckedAt,
	}
}

func (s *Service) ProbeController() map[string]any {
	s.detectController(true)
	_ = s.persistState()
	return map[string]any{
		"ok":         s.state.Controller != nil && s.state.Controller.Reachable,
		"controller": s.GetControllerStatus(),
	}
}

func (s *Service) ReloadConfig() map[string]any {
	_ = s.LoadAll()
	s.pushEvent("info", "reload", "配置已重新加载并渲染")
	runtimeApply := s.applyRuntimeConfig("配置重载")
	_ = s.persistState()
	message := "配置已重新加载并应用到代理核心"
	if deferred, ok := runtimeApply["deferred"].(bool); ok && deferred {
		message = fmt.Sprintf("配置已重新加载：%v", runtimeApply["message"])
	} else if ok, ok2 := runtimeApply["ok"].(bool); ok2 && !ok {
		message = fmt.Sprintf("配置已重新加载，但应用到代理核心失败：%v", runtimeApply["message"])
	}
	return map[string]any{"ok": true, "message": message, "runtime": s.state.Runtime, "runtimeApply": runtimeApply}
}

func (s *Service) readControllerResultMessage(result *controllerResult, fallback string) string {
	if result == nil {
		return fallback
	}
	switch payload := result.Payload.(type) {
	case string:
		if strings.TrimSpace(payload) != "" {
			return strings.TrimSpace(payload)
		}
	case map[string]any:
		if value, ok := payload["message"].(string); ok && value != "" {
			return value
		}
		if value, ok := payload["error"].(string); ok && value != "" {
			return value
		}
	}
	return fallback
}

func (s *Service) applyRuntimeConfig(reason string) map[string]any {
	appliedAt := nowISO()
	if s.config.Runtime.RenderOnly {
		message := "当前只保存配置，暂不会自动应用到代理核心。"
		s.state.Runtime.LastApplyAt = appliedAt
		s.state.Runtime.LastApplyStatus = "deferred"
		s.state.Runtime.LastApplyMessage = message
		_ = s.persistState()
		return map[string]any{"ok": false, "deferred": true, "message": message}
	}

	s.detectController(false)
	result, err := s.controller.ReloadConfig(s.layout.MihomoConfigPath)
	if err != nil {
		message := err.Error()
		s.state.Runtime.LastApplyAt = appliedAt
		s.state.Runtime.LastApplyStatus = "failed"
		s.state.Runtime.LastApplyMessage = message
		s.pushEvent("error", "runtime-apply", fmt.Sprintf("%s: %s", reason, message))
		_ = s.persistState()
		return map[string]any{"ok": false, "message": message}
	}

	message := s.readControllerResultMessage(result, fmt.Sprintf("代理核心返回状态码 %d", result.Status))
	if result.OK {
		message = "最新配置已发送到代理核心。"
		s.syncTransitMirrorSelections("", "")
	}
	s.state.Runtime.LastApplyAt = appliedAt
	if result.OK {
		s.state.Runtime.LastApplyStatus = "success"
		s.pushEvent("info", "runtime-apply", fmt.Sprintf("%s: %s", reason, message))
	} else {
		s.state.Runtime.LastApplyStatus = "failed"
		s.pushEvent("warn", "runtime-apply", fmt.Sprintf("%s: %s", reason, message))
	}
	s.state.Runtime.LastApplyMessage = message
	_ = s.persistState()
	return map[string]any{"ok": result.OK, "status": result.Status, "message": message}
}

func (s *Service) RefreshProvider(providerName string) map[string]any {
	var subscription *Subscription
	for index := range s.config.Subscriptions {
		if s.config.Subscriptions[index].Name == providerName {
			subscription = &s.config.Subscriptions[index]
			break
		}
	}
	if subscription == nil {
		return map[string]any{"ok": false, "error": fmt.Sprintf("未找到订阅：%s", providerName)}
	}
	record, ok, statusCode, statusText, err := FetchProviderSnapshot(*subscription)
	if err != nil {
		existing := s.getProviderRecord(providerName)
		existing.RefreshedAt = nowISO()
		existing.LastError = err.Error()
		s.state.Providers[providerName] = existing
		s.pushEvent("error", "provider", fmt.Sprintf("%s 刷新失败", providerName))
		_ = s.persistState()
		return map[string]any{"ok": false, "error": err.Error(), "provider": existing}
	}
	if !ok {
		record.LastError = fmt.Sprintf("%d %s", statusCode, statusText)
	}
	s.state.Providers[providerName] = *record
	runtimePayload := map[string]any{}
	if isManualProviderType(subscription.Type) {
		runtimePayload["ok"] = false
		runtimePayload["deferred"] = true
		runtimePayload["message"] = "手动节点无需远程刷新代理核心订阅。"
	} else {
		runtimeRefresh, runtimeErr := s.controller.RefreshProvider(providerName)
		if runtimeErr != nil {
			runtimePayload["ok"] = false
			runtimePayload["error"] = runtimeErr.Error()
		} else {
			runtimePayload["ok"] = runtimeRefresh.OK
			runtimePayload["status"] = runtimeRefresh.Status
			runtimePayload["payload"] = runtimeRefresh.Payload
		}
	}
	s.pushEvent(ternaryString(ok, "info", "warn"), "provider", fmt.Sprintf("%s 刷新完成，节点数 %d", providerName, record.NodeCount))
	_ = s.persistState()
	return map[string]any{"ok": ok, "provider": s.state.Providers[providerName], "runtime": runtimePayload}
}

func (s *Service) SelectGroup(groupName, proxyName string) (map[string]any, int) {
	var group *EgressGroup
	for index := range s.config.EgressGroups {
		if s.config.EgressGroups[index].Name == groupName {
			group = &s.config.EgressGroups[index]
			break
		}
	}
	if group == nil {
		return map[string]any{"ok": false, "error": fmt.Sprintf("未找到出口线路：%s", groupName)}, 404
	}
	view := s.buildGroupView(*group)
	found := false
	runtimeProxyName := proxyName
	selectedValue := proxyName
	for _, candidate := range view.Candidates {
		if candidate.ID == proxyName || (group.LandingProxy != "" && candidate.NodeName == proxyName) {
			found = true
			selectedValue = candidate.ID
			runtimeProxyName = firstNonEmpty(candidate.NodeName, candidate.ID)
			break
		}
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("你选择的节点不在出口线路“%s”里", groupName)}, 400
	}

	s.state.GroupSelections[groupName] = selectedValue
	runtimeGroupName := groupName
	if group.LandingProxy != "" {
		runtimeGroupName = relaySourceGroupName(group.Name)
	}
	runtimeResult, err := s.controller.SelectProxy(runtimeGroupName, runtimeProxyName)
	runtimePayload := map[string]any{}
	if err != nil {
		runtimePayload["ok"] = false
		runtimePayload["error"] = err.Error()
	} else {
		runtimePayload["ok"] = runtimeResult.OK
		runtimePayload["status"] = runtimeResult.Status
		runtimePayload["payload"] = runtimeResult.Payload
	}
	s.pushEvent("info", "group", fmt.Sprintf("出口线路 %s 已切换到 %s", groupName, selectedValue))
	s.syncTransitMirrorSelections(groupName, runtimeProxyName)
	_ = s.persistState()
	result := s.buildGroupView(*group)
	return map[string]any{"ok": true, "group": result, "runtime": runtimePayload}, 200
}

func (s *Service) RunHealthcheck(groupName string) (map[string]any, int) {
	var group *EgressGroup
	for index := range s.config.EgressGroups {
		if s.config.EgressGroups[index].Name == groupName {
			group = &s.config.EgressGroups[index]
			break
		}
	}
	if group == nil {
		return map[string]any{"ok": false, "error": fmt.Sprintf("未找到出口线路：%s", groupName)}, 404
	}
	s.state.Groups[groupName] = GroupState{LastHealthcheckAt: nowISO()}
	s.pushEvent("info", "healthcheck", fmt.Sprintf("已执行出口线路健康检查：%s", groupName))
	_ = s.persistState()
	return map[string]any{"ok": true, "group": s.buildGroupView(*group)}, 200
}

func extractHealthcheckDelay(payload any) int {
	switch value := payload.(type) {
	case map[string]any:
		for _, key := range []string{"delay", "meanDelay"} {
			switch delay := value[key].(type) {
			case float64:
				return int(delay)
			case int:
				return delay
			}
		}
	case []any:
		best := 0
		for _, item := range value {
			delay := extractHealthcheckDelay(item)
			if delay > 0 && (best == 0 || delay < best) {
				best = delay
			}
		}
		return best
	}
	return 0
}

func (s *Service) validateTransitRouteBinding(routeName string) (*TransitRouteView, string) {
	routeView := s.getTransitRouteViewByName(routeName)
	if routeView == nil {
		return nil, fmt.Sprintf("中转线路 %s 不存在", routeName)
	}
	if routeView.Status != "configured" {
		return routeView, buildTransitRouteBindingError(routeView)
	}
	return routeView, ""
}

func (s *Service) syncTransitMirrorSelections(egressGroupName, desiredProxyName string) {
	if s.config.Runtime.RenderOnly {
		return
	}

	for _, route := range s.config.TransitRoutes {
		if !isTransitRouteEnabled(route) {
			continue
		}
		if egressGroupName != "" && route.EgressGroup != egressGroupName {
			continue
		}

		var targetGroup *EgressGroup
		for index := range s.config.EgressGroups {
			if s.config.EgressGroups[index].Name == route.EgressGroup {
				targetGroup = &s.config.EgressGroups[index]
				break
			}
		}
		if targetGroup == nil || firstNonEmpty(targetGroup.Mode, "manual") != "manual" {
			continue
		}

		routeView, validationError := s.validateTransitRouteBinding(route.Name)
		if validationError != "" || routeView == nil {
			continue
		}

		proxyName := desiredProxyName
		if proxyName == "" {
			targetView := s.buildGroupView(*targetGroup)
			proxyName = targetView.Current
		}
		if strings.TrimSpace(proxyName) == "" {
			continue
		}

		runtimeGroupName := routeView.RuntimeGroupName
		if targetGroup.LandingProxy != "" {
			runtimeGroupName = relaySourceGroupName(runtimeGroupName)
		}
		result, err := s.controller.SelectProxy(runtimeGroupName, proxyName)
		if err != nil {
			s.pushEvent("warn", "transit-sync", fmt.Sprintf("中转线路 %s 的镜像组同步失败: %s", route.Name, err.Error()))
			continue
		}
		if !result.OK {
			s.pushEvent("warn", "transit-sync", fmt.Sprintf("中转线路 %s 的镜像组同步失败: %s", route.Name, s.readControllerResultMessage(result, fmt.Sprintf("代理核心返回状态码 %d", result.Status))))
		}
	}
}

func decodeIntoConfig(payload map[string]any) (*Config, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func DecodeConfigPayload(payload map[string]any) (*Config, error) {
	return decodeIntoConfig(payload)
}

func (s *Service) AddSubscription(payload map[string]any) (map[string]any, int) {
	providerType := firstNonEmpty(stringValue(payload["type"]), "mihomo-http")
	name := strings.TrimSpace(stringValue(payload["name"]))
	urlValue := strings.TrimSpace(stringValue(payload["url"]))
	server := strings.TrimSpace(stringValue(payload["server"]))
	port := intValue(payload["port"], 0)
	username := strings.TrimSpace(stringValue(payload["username"]))
	password := stringValue(payload["password"])

	if importText := strings.TrimSpace(stringValue(payload["import_text"])); importText != "" {
		parsed, err := parseManualProviderInput(importText, providerType)
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}, 400
		}
		if parsed.Type != "" {
			providerType = parsed.Type
		}
		if parsed.Server != "" {
			server = parsed.Server
		}
		if parsed.Port > 0 {
			port = parsed.Port
		}
		if parsed.Username != "" {
			username = parsed.Username
		}
		if parsed.Password != "" {
			password = parsed.Password
		}
	}

	if isManualProviderType(providerType) {
		if name == "" {
			if server == "" || port <= 0 {
				return map[string]any{"ok": false, "error": "手动节点的名称为空，且无法根据地址和端口自动生成"}, 400
			}
			name = buildManualProviderName(providerType, server, port)
		}
		if server == "" || port <= 0 {
			return map[string]any{"ok": false, "error": "手动节点的地址和端口不能为空"}, 400
		}
	} else if name == "" || urlValue == "" {
		return map[string]any{"ok": false, "error": "订阅名称和 URL 不能为空"}, 400
	}

	for _, subscription := range s.config.Subscriptions {
		if subscription.Name == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("订阅 %s 已存在", name)}, 400
		}
	}
	next := mustJSONClone(*s.config)
	next.Subscriptions = append(next.Subscriptions, Subscription{
		Name:                name,
		Type:                providerType,
		URL:                 urlValue,
		Server:              server,
		Port:                port,
		Username:            username,
		Password:            password,
		Interval:            intValue(payload["interval"], 3600),
		Enabled:             boolValue(payload["enabled"], true),
		HealthCheckURL:      firstNonEmpty(stringValue(payload["health_check_url"]), "https://www.gstatic.com/generate_204"),
		HealthCheckInterval: intValue(payload["health_check_interval"], 300),
	})
	result, status := s.SaveConfig(&next)
	if status == 200 {
		s.RefreshProvider(name)
	}
	return result, status
}

func stringValue(input any) string {
	if value, ok := input.(string); ok {
		return value
	}
	return ""
}

func intValue(input any, fallback int) int {
	switch value := input.(type) {
	case float64:
		return int(value)
	case int:
		return value
	}
	return fallback
}

func boolValue(input any, fallback bool) bool {
	if value, ok := input.(bool); ok {
		return value
	}
	return fallback
}

func stringArrayValue(input any) []string {
	items, ok := input.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			values = append(values, value)
		}
	}
	return values
}

func (s *Service) RemoveSubscription(name string) (map[string]any, int) {
	for _, route := range s.config.TransitRoutes {
		if route.UpstreamProvider == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("订阅 %s 仍被中转线路 %s 使用，请先解除绑定", name, route.Name)}, 400
		}
	}
	found := false
	next := mustJSONClone(*s.config)
	filtered := make([]Subscription, 0, len(next.Subscriptions))
	for _, subscription := range next.Subscriptions {
		if subscription.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, subscription)
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("订阅 %s 不存在", name)}, 404
	}
	next.Subscriptions = filtered
	delete(s.state.Providers, name)
	result, status := s.SaveConfig(&next)
	if status != 200 {
		return result, status
	}
	if err := s.removeManualProviderFile(name); err != nil {
		return map[string]any{"ok": false, "error": "清理手动节点文件失败: " + err.Error()}, 500
	}
	return result, status
}

func (s *Service) ToggleSubscription(name string, enabled bool) (map[string]any, int) {
	next := mustJSONClone(*s.config)
	found := false
	for index := range next.Subscriptions {
		if next.Subscriptions[index].Name == name {
			next.Subscriptions[index].Enabled = enabled
			found = true
			break
		}
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("订阅 %s 不存在", name)}, 404
	}
	return s.SaveConfig(&next)
}

func (s *Service) UpdateSubscription(name string, payload map[string]any) (map[string]any, int) {
	next := mustJSONClone(*s.config)
	index := -1
	for i := range next.Subscriptions {
		if next.Subscriptions[i].Name == name {
			index = i
			break
		}
	}
	if index < 0 {
		return map[string]any{"ok": false, "error": fmt.Sprintf("订阅 %s 不存在", name)}, 404
	}

	current := next.Subscriptions[index]
	if !isManualProviderType(current.Type) {
		return map[string]any{"ok": false, "error": "当前只支持编辑手动节点"}, 400
	}

	nextType := current.Type
	if payload["type"] != nil {
		nextType = strings.TrimSpace(stringValue(payload["type"]))
	}
	nextName := current.Name
	if payload["name"] != nil {
		nextName = strings.TrimSpace(stringValue(payload["name"]))
	}
	nextServer := current.Server
	if payload["server"] != nil {
		nextServer = strings.TrimSpace(stringValue(payload["server"]))
	}
	nextPort := current.Port
	if payload["port"] != nil {
		nextPort = intValue(payload["port"], 0)
	}
	nextUsername := current.Username
	if payload["username"] != nil {
		nextUsername = strings.TrimSpace(stringValue(payload["username"]))
	}
	nextPassword := current.Password
	if payload["password"] != nil {
		nextPassword = stringValue(payload["password"])
	}

	if importText := strings.TrimSpace(stringValue(payload["import_text"])); importText != "" {
		parsed, err := parseManualProviderInput(importText, nextType)
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}, 400
		}
		if parsed.Type != "" {
			nextType = parsed.Type
		}
		if parsed.Server != "" {
			nextServer = parsed.Server
		}
		if parsed.Port > 0 {
			nextPort = parsed.Port
		}
		nextUsername = parsed.Username
		nextPassword = parsed.Password
	}

	if !isManualProviderType(nextType) {
		return map[string]any{"ok": false, "error": "手动节点仅支持 HTTP、SOCKS、SOCKS5"}, 400
	}
	if nextName == "" {
		if nextServer == "" || nextPort <= 0 {
			return map[string]any{"ok": false, "error": "手动节点的名称为空，且无法根据地址和端口自动生成"}, 400
		}
		nextName = buildManualProviderName(nextType, nextServer, nextPort)
	}
	if nextServer == "" || nextPort <= 0 {
		return map[string]any{"ok": false, "error": "手动节点的地址和端口不能为空"}, 400
	}
	for i, subscription := range next.Subscriptions {
		if i == index {
			continue
		}
		if subscription.Name == nextName {
			return map[string]any{"ok": false, "error": fmt.Sprintf("订阅 %s 已存在", nextName)}, 400
		}
	}

	oldName := current.Name
	nameChanged := oldName != nextName
	renamedSelections := map[string]string{}

	next.Subscriptions[index].Name = nextName
	next.Subscriptions[index].Type = nextType
	next.Subscriptions[index].URL = ""
	next.Subscriptions[index].Server = nextServer
	next.Subscriptions[index].Port = nextPort
	next.Subscriptions[index].Username = nextUsername
	next.Subscriptions[index].Password = nextPassword
	next.Subscriptions[index].HealthCheckURL = firstNonEmpty(stringValue(payload["health_check_url"]), current.HealthCheckURL)
	next.Subscriptions[index].HealthCheckInterval = intValue(payload["health_check_interval"], current.HealthCheckInterval)
	next.Subscriptions[index].Enabled = boolValue(payload["enabled"], current.Enabled)
	next.Subscriptions[index].Interval = max(intValue(payload["interval"], current.Interval), 1)

	if nameChanged {
		for i := range next.EgressGroups {
			if next.EgressGroups[i].Provider == oldName {
				next.EgressGroups[i].Provider = nextName
			}
			for j := range next.EgressGroups[i].Proxies {
				if next.EgressGroups[i].Proxies[j] == oldName {
					next.EgressGroups[i].Proxies[j] = nextName
				}
			}
		}
		for i := range next.TransitRoutes {
			if next.TransitRoutes[i].UpstreamProvider == oldName {
				next.TransitRoutes[i].UpstreamProvider = nextName
				if next.TransitRoutes[i].UpstreamProxyName == oldName {
					next.TransitRoutes[i].UpstreamProxyName = nextName
				}
			}
		}
		for _, group := range next.EgressGroups {
			if group.Provider != nextName {
				continue
			}
			currentValue := s.state.GroupSelections[group.Name]
			switch {
			case currentValue == oldName:
				renamedSelections[group.Name] = nextName
			case group.LandingProxy != "" && currentValue == fmt.Sprintf("%s -> %s", oldName, group.LandingProxy):
				renamedSelections[group.Name] = fmt.Sprintf("%s -> %s", nextName, group.LandingProxy)
			}
		}
	}

	oldRecord, hadRecord := s.state.Providers[oldName]
	if nameChanged {
		delete(s.state.Providers, oldName)
	}

	result, status := s.SaveConfig(&next)
	if status != 200 {
		if hadRecord {
			s.state.Providers[oldName] = oldRecord
		}
		return result, status
	}
	if nameChanged {
		if err := s.removeManualProviderFile(oldName); err != nil {
			return map[string]any{"ok": false, "error": "清理旧手动节点文件失败: " + err.Error()}, 500
		}
		delete(s.state.Providers, oldName)
	}

	if hadRecord {
		oldRecord.Provider = nextName
		s.state.Providers[nextName] = oldRecord
	}
	for groupName, selection := range renamedSelections {
		s.state.GroupSelections[groupName] = selection
	}
	refresh := s.RefreshProvider(nextName)
	if refreshOK, _ := refresh["ok"].(bool); !refreshOK {
		return map[string]any{
			"ok":       true,
			"path":     result["path"],
			"config":   result["config"],
			"provider": refresh["provider"],
			"warning":  firstNonEmpty(stringValue(refresh["error"]), "手动节点已保存，但刷新快照失败"),
		}, 200
	}
	return result, status
}

func (s *Service) AddEgressGroup(payload map[string]any) (map[string]any, int) {
	name := strings.TrimSpace(stringValue(payload["name"]))
	provider := stringValue(payload["provider"])
	mode := firstNonEmpty(stringValue(payload["mode"]), "manual")
	proxies := normalizeProxyOrder(stringArrayValue(payload["proxies"]))
	if name == "" || provider == "" {
		return map[string]any{"ok": false, "error": "出口线路名称和订阅源不能为空"}, 400
	}
	for _, group := range s.config.EgressGroups {
		if group.Name == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("出口线路 %s 已存在", name)}, 400
		}
	}
	if !s.hasSubscription(provider) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("订阅源 %s 不存在", provider)}, 400
	}
	landingProxy := strings.TrimSpace(stringValue(payload["landing_proxy"]))
	if landingProxy != "" && !s.hasLandingProxy(landingProxy) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("落地节点 %s 不存在", landingProxy)}, 400
	}
	if mode == "fallback" && len(proxies) == 0 {
		return map[string]any{"ok": false, "error": "顺序容灾组至少需要选择一个节点"}, 400
	}
	next := mustJSONClone(*s.config)
	filter := stringValue(payload["filter"])
	excludeFilter := stringValue(payload["exclude_filter"])
	healthCheckURL := ""
	interval := intValue(payload["interval"], 0)
	if mode == "fallback" {
		filter = ""
		excludeFilter = ""
		healthCheckURL = firstNonEmpty(stringValue(payload["health_check_url"]), "https://www.gstatic.com/generate_204")
		interval = max(interval, 60)
	}
	next.EgressGroups = append(next.EgressGroups, EgressGroup{
		Name:          name,
		Provider:      provider,
		Mode:          mode,
		Filter:        filter,
		ExcludeFilter: excludeFilter,
		Proxies: func() []string {
			if mode == "fallback" {
				return proxies
			}
			return nil
		}(),
		LandingProxy:   landingProxy,
		HealthCheckURL: healthCheckURL,
		Interval:       interval,
	})
	result, status := s.SaveConfig(&next)
	if status == 200 && mode == "fallback" && len(proxies) > 0 {
		s.state.GroupSelections[name] = proxies[0]
		_ = s.persistState()
	}
	return result, status
}

func (s *Service) hasSubscription(name string) bool {
	for _, subscription := range s.config.Subscriptions {
		if subscription.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) hasLandingProxy(name string) bool {
	for _, landing := range s.config.LandingProxies {
		if landing.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) hasTransitRoute(name string) bool {
	for _, route := range s.config.TransitRoutes {
		if route.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) UpdateEgressGroup(name string, payload map[string]any) (map[string]any, int) {
	next := mustJSONClone(*s.config)
	index := -1
	for i := range next.EgressGroups {
		if next.EgressGroups[i].Name == name {
			index = i
			break
		}
	}
	if index < 0 {
		return map[string]any{"ok": false, "error": fmt.Sprintf("出口线路 %s 不存在", name)}, 404
	}
	nextMode := firstNonEmpty(stringValue(payload["mode"]), firstNonEmpty(next.EgressGroups[index].Mode, "manual"))
	nextProxies := next.EgressGroups[index].Proxies
	if payload["proxies"] != nil {
		nextProxies = normalizeProxyOrder(stringArrayValue(payload["proxies"]))
	}
	if nextMode == "fallback" && len(nextProxies) == 0 {
		return map[string]any{"ok": false, "error": "顺序容灾组至少需要选择一个节点"}, 400
	}
	if provider := stringValue(payload["provider"]); provider != "" {
		if !s.hasSubscription(provider) {
			return map[string]any{"ok": false, "error": fmt.Sprintf("订阅源 %s 不存在", provider)}, 400
		}
		next.EgressGroups[index].Provider = provider
	}
	if landingProxy := strings.TrimSpace(stringValue(payload["landing_proxy"])); payload["landing_proxy"] != nil {
		if landingProxy != "" && !s.hasLandingProxy(landingProxy) {
			return map[string]any{"ok": false, "error": fmt.Sprintf("落地节点 %s 不存在", landingProxy)}, 400
		}
		next.EgressGroups[index].LandingProxy = landingProxy
	}
	if mode := stringValue(payload["mode"]); mode != "" {
		next.EgressGroups[index].Mode = mode
	}
	if nextMode == "fallback" {
		next.EgressGroups[index].Filter = ""
		next.EgressGroups[index].ExcludeFilter = ""
		next.EgressGroups[index].Proxies = nextProxies
		next.EgressGroups[index].HealthCheckURL = firstNonEmpty(stringValue(payload["health_check_url"]), firstNonEmpty(next.EgressGroups[index].HealthCheckURL, "https://www.gstatic.com/generate_204"))
		next.EgressGroups[index].Interval = max(intValue(payload["interval"], next.EgressGroups[index].Interval), 60)
	} else {
		next.EgressGroups[index].Proxies = nil
		if filter := stringValue(payload["filter"]); payload["filter"] != nil {
			next.EgressGroups[index].Filter = filter
		}
		if excludeFilter := stringValue(payload["exclude_filter"]); payload["exclude_filter"] != nil {
			next.EgressGroups[index].ExcludeFilter = excludeFilter
		}
	}
	result, status := s.SaveConfig(&next)
	if status == 200 {
		if nextMode == "fallback" && len(nextProxies) > 0 {
			if _, ok := s.state.GroupSelections[name]; !ok || !containsString(nextProxies, s.state.GroupSelections[name]) {
				s.state.GroupSelections[name] = nextProxies[0]
			}
		} else {
			delete(s.state.GroupSelections, name)
		}
		_ = s.persistState()
	}
	return result, status
}

func (s *Service) RemoveEgressGroup(name string) (map[string]any, int) {
	for _, listener := range s.config.Listeners {
		if getListenerRouteMode(listener) == "direct" && listener.EgressGroup == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("出口线路 %s 仍被本地入口使用，请先解除绑定", name)}, 400
		}
	}
	for _, route := range s.config.TransitRoutes {
		if route.EgressGroup == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("出口线路 %s 仍被中转线路 %s 引用，请先解除绑定", name, route.Name)}, 400
		}
	}
	next := mustJSONClone(*s.config)
	filtered := make([]EgressGroup, 0, len(next.EgressGroups))
	found := false
	for _, group := range next.EgressGroups {
		if group.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, group)
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("出口线路 %s 不存在", name)}, 404
	}
	next.EgressGroups = filtered
	result, status := s.SaveConfig(&next)
	if status == 200 {
		delete(s.state.GroupSelections, name)
		delete(s.state.Groups, name)
		_ = s.persistState()
	}
	return result, status
}

func (s *Service) AddLandingProxy(payload map[string]any) (map[string]any, int) {
	name := strings.TrimSpace(stringValue(payload["name"]))
	server := strings.TrimSpace(stringValue(payload["server"]))
	port := intValue(payload["port"], 0)
	if name == "" || server == "" || port == 0 {
		return map[string]any{"ok": false, "error": "落地节点名称、地址和端口不能为空"}, 400
	}
	if landingType := firstNonEmpty(stringValue(payload["type"]), "socks5"); landingType != "socks5" && landingType != "http" {
		return map[string]any{"ok": false, "error": "当前只支持 HTTP 和 SOCKS5 两种落地节点"}, 400
	}
	for _, landing := range s.config.LandingProxies {
		if landing.Name == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("落地节点 %s 已存在", name)}, 400
		}
	}
	next := mustJSONClone(*s.config)
	next.LandingProxies = append(next.LandingProxies, LandingProxy{
		Name:           name,
		Type:           firstNonEmpty(stringValue(payload["type"]), "socks5"),
		Server:         server,
		Port:           port,
		Username:       stringValue(payload["username"]),
		Password:       stringValue(payload["password"]),
		TLS:            boolValue(payload["tls"], false),
		SNI:            stringValue(payload["sni"]),
		SkipCertVerify: boolValue(payload["skip_cert_verify"], false),
		Enabled:        boolValue(payload["enabled"], true),
	})
	return s.SaveConfig(&next)
}

func (s *Service) UpdateLandingProxy(name string, payload map[string]any) (map[string]any, int) {
	next := mustJSONClone(*s.config)
	index := -1
	for i := range next.LandingProxies {
		if next.LandingProxies[i].Name == name {
			index = i
			break
		}
	}
	if index < 0 {
		return map[string]any{"ok": false, "error": fmt.Sprintf("落地节点 %s 不存在", name)}, 404
	}
	if landingType := stringValue(payload["type"]); landingType != "" {
		if landingType != "socks5" && landingType != "http" {
			return map[string]any{"ok": false, "error": "当前只支持 HTTP 和 SOCKS5 两种落地节点"}, 400
		}
		next.LandingProxies[index].Type = landingType
	}
	if server := strings.TrimSpace(stringValue(payload["server"])); payload["server"] != nil {
		next.LandingProxies[index].Server = server
	}
	if port := intValue(payload["port"], 0); port != 0 {
		next.LandingProxies[index].Port = port
	}
	if payload["username"] != nil {
		next.LandingProxies[index].Username = stringValue(payload["username"])
	}
	if payload["password"] != nil {
		next.LandingProxies[index].Password = stringValue(payload["password"])
	}
	if payload["tls"] != nil {
		next.LandingProxies[index].TLS = boolValue(payload["tls"], false)
	}
	if payload["sni"] != nil {
		next.LandingProxies[index].SNI = stringValue(payload["sni"])
	}
	if payload["skip_cert_verify"] != nil {
		next.LandingProxies[index].SkipCertVerify = boolValue(payload["skip_cert_verify"], false)
	}
	if payload["enabled"] != nil {
		next.LandingProxies[index].Enabled = boolValue(payload["enabled"], true)
	}
	return s.SaveConfig(&next)
}

func (s *Service) RemoveLandingProxy(name string) (map[string]any, int) {
	for _, group := range s.config.EgressGroups {
		if group.LandingProxy == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("落地节点 %s 仍被出口组 %s 使用，请先解除绑定", name, group.Name)}, 400
		}
	}
	next := mustJSONClone(*s.config)
	filtered := make([]LandingProxy, 0, len(next.LandingProxies))
	found := false
	for _, landing := range next.LandingProxies {
		if landing.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, landing)
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("落地节点 %s 不存在", name)}, 404
	}
	next.LandingProxies = filtered
	return s.SaveConfig(&next)
}

func (s *Service) AddListener(payload map[string]any) (map[string]any, int) {
	name := strings.TrimSpace(stringValue(payload["name"]))
	port := intValue(payload["port"], 0)
	transitRoute := strings.TrimSpace(stringValue(payload["transit_route"]))
	routeMode := normalizeListenerRouteMode(stringValue(payload["route_mode"]), transitRoute)
	egressGroup := strings.TrimSpace(stringValue(payload["egress_group"]))
	if name == "" || port == 0 {
		return map[string]any{"ok": false, "error": "本地入口名称和端口不能为空"}, 400
	}
	for _, listener := range s.config.Listeners {
		if listener.Name == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("本地入口 %s 已存在", name)}, 400
		}
		if listener.Port == port {
			return map[string]any{"ok": false, "error": fmt.Sprintf("端口 %d 已被占用", port)}, 400
		}
	}
	if routeMode == "transit" {
		if transitRoute == "" {
			return map[string]any{"ok": false, "error": "中转模式下必须选择中转线路"}, 400
		}
		if routeView, validationError := s.validateTransitRouteBinding(transitRoute); validationError != "" {
			return map[string]any{"ok": false, "error": validationError, "route": routeView}, 400
		}
	} else {
		if egressGroup == "" {
			return map[string]any{"ok": false, "error": "直连模式下必须选择出口线路"}, 400
		}
		if !s.hasGroup(egressGroup) {
			return map[string]any{"ok": false, "error": fmt.Sprintf("出口线路 %s 不存在", egressGroup)}, 400
		}
	}

	next := mustJSONClone(*s.config)
	next.Listeners = append(next.Listeners, Listener{
		Name:         name,
		Type:         firstNonEmpty(stringValue(payload["type"]), "socks"),
		Listen:       firstNonEmpty(stringValue(payload["listen"]), "0.0.0.0"),
		Port:         port,
		UDP:          boolValue(payload["udp"], true),
		Enabled:      boolValue(payload["enabled"], true),
		Users:        parseUsers(payload["users"]),
		RouteMode:    routeMode,
		EgressGroup:  ternaryString(routeMode == "direct", egressGroup, ""),
		TransitRoute: ternaryString(routeMode == "transit", transitRoute, ""),
	})
	return s.SaveConfig(&next)
}

func parseUsers(input any) []ListenerUser {
	items, ok := input.([]any)
	if !ok {
		return nil
	}
	users := []ListenerUser{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		users = append(users, ListenerUser{Username: stringValue(entry["username"]), Password: stringValue(entry["password"])})
	}
	return users
}

func (s *Service) hasGroup(name string) bool {
	for _, group := range s.config.EgressGroups {
		if group.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) UpdateListener(name string, payload map[string]any) (map[string]any, int) {
	next := mustJSONClone(*s.config)
	index := -1
	for i := range next.Listeners {
		if next.Listeners[i].Name == name {
			index = i
			break
		}
	}
	if index < 0 {
		return map[string]any{"ok": false, "error": fmt.Sprintf("本地入口 %s 不存在", name)}, 404
	}

	if port := intValue(payload["port"], 0); port != 0 && port != next.Listeners[index].Port {
		for i, listener := range next.Listeners {
			if i != index && listener.Port == port {
				return map[string]any{"ok": false, "error": fmt.Sprintf("端口 %d 已被占用", port)}, 400
			}
		}
		next.Listeners[index].Port = port
	}
	current := next.Listeners[index]
	nextRouteMode := normalizeListenerRouteMode(stringValue(payload["route_mode"]), firstNonEmpty(stringValue(payload["transit_route"]), current.TransitRoute))
	nextEgressGroup := current.EgressGroup
	if payload["egress_group"] != nil {
		nextEgressGroup = strings.TrimSpace(stringValue(payload["egress_group"]))
	}
	nextTransitRoute := current.TransitRoute
	if payload["transit_route"] != nil {
		nextTransitRoute = strings.TrimSpace(stringValue(payload["transit_route"]))
	}
	if nextRouteMode == "transit" {
		if nextTransitRoute == "" {
			return map[string]any{"ok": false, "error": "中转模式下必须选择中转线路"}, 400
		}
		if routeView, validationError := s.validateTransitRouteBinding(nextTransitRoute); validationError != "" {
			return map[string]any{"ok": false, "error": validationError, "route": routeView}, 400
		}
		next.Listeners[index].RouteMode = "transit"
		next.Listeners[index].TransitRoute = nextTransitRoute
		next.Listeners[index].EgressGroup = ""
	} else {
		if nextEgressGroup == "" {
			return map[string]any{"ok": false, "error": "直连模式下必须选择出口线路"}, 400
		}
		if !s.hasGroup(nextEgressGroup) {
			return map[string]any{"ok": false, "error": fmt.Sprintf("出口线路 %s 不存在", nextEgressGroup)}, 400
		}
		next.Listeners[index].RouteMode = "direct"
		next.Listeners[index].EgressGroup = nextEgressGroup
		next.Listeners[index].TransitRoute = ""
	}
	if listenerType := stringValue(payload["type"]); listenerType != "" {
		next.Listeners[index].Type = listenerType
	}
	if listen := stringValue(payload["listen"]); payload["listen"] != nil {
		next.Listeners[index].Listen = listen
	}
	if payload["udp"] != nil {
		next.Listeners[index].UDP = boolValue(payload["udp"], true)
	}
	if payload["enabled"] != nil {
		next.Listeners[index].Enabled = boolValue(payload["enabled"], true)
	}
	if payload["users"] != nil {
		next.Listeners[index].Users = parseUsers(payload["users"])
	}
	return s.SaveConfig(&next)
}

func (s *Service) RemoveListener(name string) (map[string]any, int) {
	next := mustJSONClone(*s.config)
	filtered := make([]Listener, 0, len(next.Listeners))
	found := false
	for _, listener := range next.Listeners {
		if listener.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, listener)
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("本地入口 %s 不存在", name)}, 404
	}
	next.Listeners = filtered
	return s.SaveConfig(&next)
}

func (s *Service) AddTransitRoute(payload map[string]any) (map[string]any, int) {
	name := strings.TrimSpace(stringValue(payload["name"]))
	upstreamProvider := strings.TrimSpace(stringValue(payload["upstream_provider"]))
	upstreamProxyName := strings.TrimSpace(stringValue(payload["upstream_proxy_name"]))
	egressGroup := strings.TrimSpace(stringValue(payload["egress_group"]))
	if name == "" || upstreamProvider == "" || upstreamProxyName == "" || egressGroup == "" {
		return map[string]any{"ok": false, "error": "中转线路名称、中转来源、中转节点和落地出口组不能为空"}, 400
	}
	if s.hasTransitRoute(name) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("中转线路 %s 已存在", name)}, 400
	}
	if !s.hasSubscription(upstreamProvider) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("中转来源 %s 不存在", upstreamProvider)}, 400
	}
	if !s.hasGroup(egressGroup) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("落地出口组 %s 不存在", egressGroup)}, 400
	}
	providerRecord := s.getProviderRecord(upstreamProvider)
	if len(providerRecord.Nodes) > 0 {
		found := false
		for _, node := range providerRecord.Nodes {
			if node.Name == upstreamProxyName {
				found = true
				break
			}
		}
		if !found {
			return map[string]any{"ok": false, "error": fmt.Sprintf("来源 %s 当前不存在节点 %s", upstreamProvider, upstreamProxyName)}, 400
		}
	}

	next := mustJSONClone(*s.config)
	next.TransitRoutes = append(next.TransitRoutes, TransitRoute{
		Name:              name,
		Enabled:           boolValue(payload["enabled"], true),
		UpstreamProvider:  upstreamProvider,
		UpstreamProxyName: upstreamProxyName,
		EgressGroup:       egressGroup,
		Notes:             strings.TrimSpace(stringValue(payload["notes"])),
	})
	return s.SaveConfig(&next)
}

func (s *Service) UpdateTransitRoute(name string, payload map[string]any) (map[string]any, int) {
	next := mustJSONClone(*s.config)
	index := -1
	for i := range next.TransitRoutes {
		if next.TransitRoutes[i].Name == name {
			index = i
			break
		}
	}
	if index < 0 {
		return map[string]any{"ok": false, "error": fmt.Sprintf("中转线路 %s 不存在", name)}, 404
	}

	current := next.TransitRoutes[index]
	upstreamProvider := current.UpstreamProvider
	if payload["upstream_provider"] != nil {
		upstreamProvider = strings.TrimSpace(stringValue(payload["upstream_provider"]))
	}
	upstreamProxyName := current.UpstreamProxyName
	if payload["upstream_proxy_name"] != nil {
		upstreamProxyName = strings.TrimSpace(stringValue(payload["upstream_proxy_name"]))
	}
	egressGroup := current.EgressGroup
	if payload["egress_group"] != nil {
		egressGroup = strings.TrimSpace(stringValue(payload["egress_group"]))
	}
	if upstreamProvider == "" || upstreamProxyName == "" || egressGroup == "" {
		return map[string]any{"ok": false, "error": "中转来源、中转节点和落地出口组不能为空"}, 400
	}
	if !s.hasSubscription(upstreamProvider) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("中转来源 %s 不存在", upstreamProvider)}, 400
	}
	if !s.hasGroup(egressGroup) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("落地出口组 %s 不存在", egressGroup)}, 400
	}
	providerRecord := s.getProviderRecord(upstreamProvider)
	if len(providerRecord.Nodes) > 0 {
		found := false
		for _, node := range providerRecord.Nodes {
			if node.Name == upstreamProxyName {
				found = true
				break
			}
		}
		if !found {
			return map[string]any{"ok": false, "error": fmt.Sprintf("来源 %s 当前不存在节点 %s", upstreamProvider, upstreamProxyName)}, 400
		}
	}

	next.TransitRoutes[index].Enabled = boolValue(payload["enabled"], current.Enabled)
	next.TransitRoutes[index].UpstreamProvider = upstreamProvider
	next.TransitRoutes[index].UpstreamProxyName = upstreamProxyName
	next.TransitRoutes[index].EgressGroup = egressGroup
	if payload["notes"] != nil {
		next.TransitRoutes[index].Notes = strings.TrimSpace(stringValue(payload["notes"]))
	}

	result, status := s.SaveConfig(&next)
	if status == 200 {
		delete(s.state.TransitRoutes, name)
		_ = s.persistState()
	}
	return result, status
}

func (s *Service) RemoveTransitRoute(name string) (map[string]any, int) {
	if !s.hasTransitRoute(name) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("中转线路 %s 不存在", name)}, 404
	}
	for _, listener := range s.config.Listeners {
		if getListenerRouteMode(listener) == "transit" && listener.TransitRoute == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("中转线路 %s 仍被本地入口使用，请先解除绑定", name)}, 400
		}
	}

	next := mustJSONClone(*s.config)
	filtered := make([]TransitRoute, 0, len(next.TransitRoutes))
	for _, route := range next.TransitRoutes {
		if route.Name == name {
			continue
		}
		filtered = append(filtered, route)
	}
	next.TransitRoutes = filtered
	result, status := s.SaveConfig(&next)
	if status == 200 {
		delete(s.state.TransitRoutes, name)
		_ = s.persistState()
	}
	return result, status
}

func (s *Service) RunTransitRouteHealthcheck(name string) (map[string]any, int) {
	routeView, validationError := s.validateTransitRouteBinding(name)
	if validationError != "" {
		return map[string]any{"ok": false, "error": validationError, "route": routeView}, 400
	}

	healthCheckURL := "https://www.gstatic.com/generate_204"
	for _, group := range s.config.EgressGroups {
		if group.Name == routeView.EgressGroup && group.HealthCheckURL != "" {
			healthCheckURL = group.HealthCheckURL
			break
		}
	}

	startedAt := nowISO()
	s.state.TransitRoutes[name] = TransitRouteState{
		LastTestedAt:    startedAt,
		LastTestStatus:  "running",
		LastTestMessage: "检测中",
		LastTestURL:     healthCheckURL,
	}
	_ = s.persistState()

	result, err := s.controller.HealthcheckGroup(routeView.RuntimeGroupName, healthCheckURL, 5000)
	runtimePayload := map[string]any{}
	ok := false
	message := ""
	delay := 0
	statusCode := 0
	if err != nil {
		message = err.Error()
		runtimePayload["ok"] = false
		runtimePayload["error"] = message
	} else {
		ok = result.OK
		statusCode = result.Status
		message = s.readControllerResultMessage(result, "中转线路检测失败")
		if ok {
			message = "中转线路检测完成"
			delay = extractHealthcheckDelay(result.Payload)
		}
		runtimePayload["ok"] = result.OK
		runtimePayload["status"] = result.Status
		runtimePayload["payload"] = result.Payload
	}

	s.state.TransitRoutes[name] = TransitRouteState{
		LastTestedAt:    nowISO(),
		LastTestStatus:  ternaryString(ok, "success", "failed"),
		LastTestMessage: message,
		LastTestDelay:   delay,
		LastTestURL:     healthCheckURL,
	}
	s.pushEvent(ternaryString(ok, "info", "warn"), "transit-healthcheck", fmt.Sprintf("已执行中转线路检测：%s (%s)", name, firstNonEmpty(message, fmt.Sprintf("状态码 %d", statusCode))))
	_ = s.persistState()

	refreshedRoute := s.getTransitRouteViewByName(name)
	if !ok {
		return map[string]any{"ok": false, "error": message, "route": refreshedRoute, "runtime": runtimePayload}, 400
	}
	return map[string]any{"ok": true, "route": refreshedRoute, "runtime": runtimePayload}, 200
}

func (s *Service) MihomoVersions() (MihomoVersionsResponse, error) {
	records, err := s.store.ListVersionRecords()
	if err != nil {
		return MihomoVersionsResponse{}, err
	}
	response := MihomoVersionsResponse{
		Platform: map[string]string{
			"os":   runtime.GOOS,
			"arch": runtime.GOARCH,
		},
		Recommended:       "v1.19.21",
		ConfiguredBinary:  s.config.Runtime.MihomoBinary,
		SupportMatrix:     defaultMihomoSupportMatrix(),
		InstalledVersions: records,
	}
	for _, record := range records {
		if record.ActivatedAt != "" {
			response.ActiveVersion = record.Version
			break
		}
	}
	return response, nil
}

func (s *Service) MihomoVersionState() (map[string]any, error) {
	records, err := s.store.ListVersionRecords()
	if err != nil {
		return nil, err
	}
	return map[string]any{"records": records}, nil
}

func (s *Service) findSupportedVersion(version string) (MihomoVersionSupport, bool) {
	for _, item := range defaultMihomoSupportMatrix() {
		if item.Version == version {
			return item, true
		}
	}
	return MihomoVersionSupport{}, false
}

func (s *Service) DownloadMihomoVersion(version string) (map[string]any, int) {
	support, ok := s.findSupportedVersion(version)
	if !ok || !support.Supported || support.AssetURL == "" {
		return map[string]any{"ok": false, "error": fmt.Sprintf("版本 %s 不在当前支持列表中", version)}, 400
	}
	versionDir := filepath.Join(s.layout.MihomoVersionsDir, version)
	archivePath := filepath.Join(versionDir, support.AssetName)
	if err := downloadFile(archivePath, support.AssetURL); err != nil {
		record := MihomoVersionRecord{Version: version, Status: "failed", ArchivePath: archivePath, LastError: err.Error()}
		_ = s.store.UpsertVersionRecord(record)
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	record := MihomoVersionRecord{Version: version, Status: "downloaded", ArchivePath: archivePath, DownloadedAt: nowISO()}
	_ = s.store.UpsertVersionRecord(record)
	s.pushEvent("info", "mihomo", fmt.Sprintf("Mihomo %s 已下载", version))
	_ = s.persistState()
	return map[string]any{"ok": true, "record": record}, 200
}

func (s *Service) InstallMihomoVersion(version string) (map[string]any, int) {
	records, err := s.store.ListVersionRecords()
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	var record *MihomoVersionRecord
	for index := range records {
		if records[index].Version == version {
			record = &records[index]
			break
		}
	}
	if record == nil || record.ArchivePath == "" {
		return map[string]any{"ok": false, "error": "请先下载该版本"}, 400
	}
	binaryName := "mihomo"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(s.layout.MihomoVersionsDir, version, binaryName)
	if strings.HasSuffix(record.ArchivePath, ".gz") {
		if err := extractGzipArchive(record.ArchivePath, binaryPath); err != nil {
			record.Status = "failed"
			record.LastError = err.Error()
			_ = s.store.UpsertVersionRecord(*record)
			return map[string]any{"ok": false, "error": err.Error()}, 500
		}
	} else {
		return map[string]any{"ok": false, "error": "当前仅支持 .gz 制品自动安装"}, 400
	}
	record.Status = "installed"
	record.BinaryPath = binaryPath
	record.InstalledAt = nowISO()
	record.LastError = ""
	_ = s.store.UpsertVersionRecord(*record)
	s.pushEvent("info", "mihomo", fmt.Sprintf("Mihomo %s 已安装", version))
	_ = s.persistState()
	return map[string]any{"ok": true, "record": record}, 200
}

func (s *Service) ActivateMihomoVersion(version string) (map[string]any, int) {
	records, err := s.store.ListVersionRecords()
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	targetIndex := -1
	for index := range records {
		if records[index].Version == version {
			targetIndex = index
		} else if records[index].ActivatedAt != "" {
			records[index].ActivatedAt = ""
			_ = s.store.UpsertVersionRecord(records[index])
		}
	}
	if targetIndex < 0 || records[targetIndex].BinaryPath == "" {
		return map[string]any{"ok": false, "error": "目标版本尚未安装"}, 400
	}

	next := mustJSONClone(*s.config)
	next.Runtime.MihomoBinary = records[targetIndex].BinaryPath
	result, status := s.SaveConfig(&next)
	if status != 200 {
		return result, status
	}

	records[targetIndex].ActivatedAt = nowISO()
	records[targetIndex].Status = "active"
	_ = s.store.UpsertVersionRecord(records[targetIndex])
	s.pushEvent("info", "mihomo", fmt.Sprintf("Mihomo 已切换到 %s", version))
	_ = s.persistState()
	return map[string]any{"ok": true, "record": records[targetIndex]}, 200
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func (s *Service) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

func (s *Service) Config() *Config {
	return s.config
}

func (s *Service) RefreshIfConfigChanged() (bool, error) {
	if s == nil || strings.TrimSpace(s.configPath) == "" {
		return false, nil
	}
	stat, err := os.Stat(s.configPath)
	if err != nil {
		return false, err
	}
	if !stat.ModTime().Equal(s.configMTime) {
		return true, s.LoadAll()
	}
	return false, nil
}

func (s *Service) AuthFingerprint() string {
	if s == nil || s.config == nil || s.auth == nil {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		s.config.Admin.Username,
		s.config.Admin.PasswordHash,
		s.config.Admin.Password,
		s.auth.secret,
	}, "|")))
	return fmt.Sprintf("%x", sum[:])
}

func (s *Service) captureConfigModTime() error {
	stat, err := os.Stat(s.configPath)
	if err != nil {
		return err
	}
	s.configMTime = stat.ModTime()
	return nil
}
