package axis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const defaultConfigPath = "config/proxyrelay.yaml"

type Service struct {
	configPath string
	config     *Config
	layout     *RuntimeLayout
	state      *AppState
	store      *Store
	auth       *AuthContext
	controller *MihomoControllerAdapter
}

func NewService(configPath string) (*Service, error) {
	if configPath == "" {
		configPath = defaultConfigPath
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

	if err := s.loadState(); err != nil {
		return err
	}
	s.detectRuntime()
	if _, err := s.renderRuntimeConfig(); err != nil {
		return err
	}
	s.detectController(false)
	return s.persistState()
}

func (s *Service) loadState() error {
	state, err := s.store.LoadState()
	if err != nil {
		return err
	}
	if state != nil {
		s.state = state
		s.state.Events, _ = s.store.ListEvents(100)
		return nil
	}

	if fileExists(s.layout.StatePath) {
		raw, err := os.ReadFile(s.layout.StatePath)
		if err == nil {
			var loaded AppState
			if json.Unmarshal(raw, &loaded) == nil {
				s.state = &loaded
				if s.state.Providers == nil {
					s.state.Providers = map[string]ProviderRecord{}
				}
				if s.state.GroupSelections == nil {
					s.state.GroupSelections = map[string]string{}
				}
				if s.state.Groups == nil {
					s.state.Groups = map[string]GroupState{}
				}
				s.state.Events, _ = s.store.ListEvents(100)
				return nil
			}
		}
	}

	s.state = createEmptyState()
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
	s.state.Runtime.LastRenderAt = nowISO()
	s.pushEvent("info", "render", "最新代理核心配置已生成")
	return rendered, nil
}

func (s *Service) Login(username, password string) (map[string]any, int) {
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

func BuildSetupState(config *Config) SetupState {
	hasSubscriptions := len(config.Subscriptions) > 0
	hasRealSubscriptions := false
	for _, subscription := range config.Subscriptions {
		if !isPlaceholderSubscriptionURL(subscription.URL) {
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

func (s *Service) GetStatus() map[string]any {
	s.detectController(false)
	providers := s.GetProviders()
	groups := s.GetGroups()
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
			"providers": len(providers),
			"groups":    len(groups),
			"listeners": len(listeners),
			"nodes":     sumProviderNodes(providers),
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
		items = append(items, map[string]any{
			"name":        subscription.Name,
			"type":        firstNonEmpty(subscription.Type, "mihomo-http"),
			"urlMasked":   firstNonEmpty(record.URLMasked, subscription.URL),
			"interval":    max(subscription.Interval, 3600),
			"enabled":     subscription.Enabled,
			"refreshedAt": record.RefreshedAt,
			"nodeCount":   record.NodeCount,
			"lastError":   record.LastError,
			"nodes":       record.Nodes,
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

func (s *Service) buildGroupView(group EgressGroup) GroupView {
	providerRecord := s.getProviderRecord(group.Provider)
	candidates := []GroupCandidate{}
	for _, node := range providerRecord.Nodes {
		if matchesGroup(group, node) {
			candidates = append(candidates, GroupCandidate{ID: node.ID, Name: node.Name, Type: node.Type, Server: node.Server, Port: node.Port})
		}
	}
	selection := s.state.GroupSelections[group.Name]
	if selection == "" && len(candidates) > 0 {
		selection = candidates[0].Name
	}
	groupState := s.state.Groups[group.Name]
	return GroupView{
		Name:              group.Name,
		Mode:              firstNonEmpty(group.Mode, "manual"),
		Provider:          group.Provider,
		Filter:            group.Filter,
		CandidateCount:    len(candidates),
		Current:           selection,
		Candidates:        candidates,
		LastHealthcheckAt: groupState.LastHealthcheckAt,
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

	views := make([]GroupView, 0, len(s.config.EgressGroups))
	for _, group := range s.config.EgressGroups {
		view := s.buildGroupView(group)
		_, providerExists := providerNames[group.Provider]
		_, providerEnabled := enabledProviders[group.Provider]
		view.ProviderMissing = !providerExists
		view.ProviderDisabled = providerExists && !providerEnabled
		views = append(views, view)
	}
	return views
}

func (s *Service) GetListeners() []ListenerView {
	groupMap := map[string]GroupView{}
	for _, group := range s.GetGroups() {
		groupMap[group.Name] = group
	}
	groupNames := map[string]struct{}{}
	for _, group := range s.config.EgressGroups {
		groupNames[group.Name] = struct{}{}
	}

	listeners := make([]ListenerView, 0, len(s.config.Listeners))
	for _, listener := range s.config.Listeners {
		group, ok := groupMap[listener.EgressGroup]
		groupMissing := false
		if _, exists := groupNames[listener.EgressGroup]; !exists {
			groupMissing = true
		}
		status := "configured"
		if groupMissing {
			status = "orphaned"
		} else if group.ProviderMissing || group.ProviderDisabled {
			status = "degraded"
		}

		listeners = append(listeners, ListenerView{
			Name:             listener.Name,
			Type:             firstNonEmpty(listener.Type, "socks"),
			Listen:           firstNonEmpty(listener.Listen, "0.0.0.0"),
			Port:             listener.Port,
			UDP:              listener.UDP,
			Users:            listener.Users,
			UserCount:        len(listener.Users),
			EgressGroup:      listener.EgressGroup,
			CurrentProxy:     ternaryString(ok, group.Current, ""),
			Status:           status,
			GroupMissing:     groupMissing,
			ProviderMissing:  group.ProviderMissing,
			ProviderDisabled: group.ProviderDisabled,
		})
	}
	return listeners
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
	runtimeRefresh, runtimeErr := s.controller.RefreshProvider(providerName)
	runtimePayload := map[string]any{}
	if runtimeErr != nil {
		runtimePayload["ok"] = false
		runtimePayload["error"] = runtimeErr.Error()
	} else {
		runtimePayload["ok"] = runtimeRefresh.OK
		runtimePayload["status"] = runtimeRefresh.Status
		runtimePayload["payload"] = runtimeRefresh.Payload
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
		return map[string]any{"ok": false, "error": fmt.Sprintf("未找到出口组: %s", groupName)}, 404
	}
	view := s.buildGroupView(*group)
	found := false
	for _, candidate := range view.Candidates {
		if candidate.Name == proxyName {
			found = true
			break
		}
	}
	if !found {
		return map[string]any{"ok": false, "error": fmt.Sprintf("节点不属于出口组 %s", groupName)}, 400
	}

	s.state.GroupSelections[groupName] = proxyName
	runtimeResult, err := s.controller.SelectProxy(groupName, proxyName)
	runtimePayload := map[string]any{}
	if err != nil {
		runtimePayload["ok"] = false
		runtimePayload["error"] = err.Error()
	} else {
		runtimePayload["ok"] = runtimeResult.OK
		runtimePayload["status"] = runtimeResult.Status
		runtimePayload["payload"] = runtimeResult.Payload
	}
	s.pushEvent("info", "group", fmt.Sprintf("出口组 %s 已切换到 %s", groupName, proxyName))
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
		return map[string]any{"ok": false, "error": fmt.Sprintf("未找到出口组: %s", groupName)}, 404
	}
	s.state.Groups[groupName] = GroupState{LastHealthcheckAt: nowISO()}
	s.pushEvent("info", "healthcheck", fmt.Sprintf("已执行出口组健康检查: %s", groupName))
	_ = s.persistState()
	return map[string]any{"ok": true, "group": s.buildGroupView(*group)}, 200
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

func (s *Service) AddSubscription(payload map[string]any) (map[string]any, int) {
	name, _ := payload["name"].(string)
	urlValue, _ := payload["url"].(string)
	if strings.TrimSpace(name) == "" || strings.TrimSpace(urlValue) == "" {
		return map[string]any{"ok": false, "error": "订阅名称和 URL 不能为空"}, 400
	}
	for _, subscription := range s.config.Subscriptions {
		if subscription.Name == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("订阅 %s 已存在", name)}, 400
		}
	}
	next := mustJSONClone(*s.config)
	next.Subscriptions = append(next.Subscriptions, Subscription{
		Name:                strings.TrimSpace(name),
		Type:                firstNonEmpty(stringValue(payload["type"]), "mihomo-http"),
		URL:                 strings.TrimSpace(urlValue),
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

func (s *Service) RemoveSubscription(name string) (map[string]any, int) {
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
	return s.SaveConfig(&next)
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

func (s *Service) AddEgressGroup(payload map[string]any) (map[string]any, int) {
	name := strings.TrimSpace(stringValue(payload["name"]))
	provider := stringValue(payload["provider"])
	if name == "" || provider == "" {
		return map[string]any{"ok": false, "error": "出口组名称和订阅源不能为空"}, 400
	}
	for _, group := range s.config.EgressGroups {
		if group.Name == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("出口组 %s 已存在", name)}, 400
		}
	}
	if !s.hasSubscription(provider) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("订阅源 %s 不存在", provider)}, 400
	}
	next := mustJSONClone(*s.config)
	next.EgressGroups = append(next.EgressGroups, EgressGroup{Name: name, Provider: provider, Mode: firstNonEmpty(stringValue(payload["mode"]), "manual"), Filter: stringValue(payload["filter"]), ExcludeFilter: stringValue(payload["exclude_filter"])})
	return s.SaveConfig(&next)
}

func (s *Service) hasSubscription(name string) bool {
	for _, subscription := range s.config.Subscriptions {
		if subscription.Name == name {
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
		return map[string]any{"ok": false, "error": fmt.Sprintf("出口组 %s 不存在", name)}, 404
	}
	if provider := stringValue(payload["provider"]); provider != "" {
		if !s.hasSubscription(provider) {
			return map[string]any{"ok": false, "error": fmt.Sprintf("订阅源 %s 不存在", provider)}, 400
		}
		next.EgressGroups[index].Provider = provider
	}
	if mode := stringValue(payload["mode"]); mode != "" {
		next.EgressGroups[index].Mode = mode
	}
	if filter := stringValue(payload["filter"]); payload["filter"] != nil {
		next.EgressGroups[index].Filter = filter
	}
	if excludeFilter := stringValue(payload["exclude_filter"]); payload["exclude_filter"] != nil {
		next.EgressGroups[index].ExcludeFilter = excludeFilter
	}
	return s.SaveConfig(&next)
}

func (s *Service) RemoveEgressGroup(name string) (map[string]any, int) {
	for _, listener := range s.config.Listeners {
		if listener.EgressGroup == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("出口组 %s 仍被监听接口引用，请先删除相关接口", name)}, 400
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
		return map[string]any{"ok": false, "error": fmt.Sprintf("出口组 %s 不存在", name)}, 404
	}
	next.EgressGroups = filtered
	return s.SaveConfig(&next)
}

func (s *Service) AddListener(payload map[string]any) (map[string]any, int) {
	name := strings.TrimSpace(stringValue(payload["name"]))
	port := intValue(payload["port"], 0)
	egressGroup := stringValue(payload["egress_group"])
	if name == "" || port == 0 || egressGroup == "" {
		return map[string]any{"ok": false, "error": "监听名称、端口和出口组不能为空"}, 400
	}
	for _, listener := range s.config.Listeners {
		if listener.Name == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("监听 %s 已存在", name)}, 400
		}
		if listener.Port == port {
			return map[string]any{"ok": false, "error": fmt.Sprintf("端口 %d 已被占用", port)}, 400
		}
	}
	if !s.hasGroup(egressGroup) {
		return map[string]any{"ok": false, "error": fmt.Sprintf("出口组 %s 不存在", egressGroup)}, 400
	}

	next := mustJSONClone(*s.config)
	next.Listeners = append(next.Listeners, Listener{
		Name:        name,
		Type:        firstNonEmpty(stringValue(payload["type"]), "socks"),
		Listen:      firstNonEmpty(stringValue(payload["listen"]), "0.0.0.0"),
		Port:        port,
		UDP:         boolValue(payload["udp"], true),
		Enabled:     boolValue(payload["enabled"], true),
		Users:       parseUsers(payload["users"]),
		EgressGroup: egressGroup,
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
		return map[string]any{"ok": false, "error": fmt.Sprintf("监听 %s 不存在", name)}, 404
	}

	if port := intValue(payload["port"], 0); port != 0 && port != next.Listeners[index].Port {
		for i, listener := range next.Listeners {
			if i != index && listener.Port == port {
				return map[string]any{"ok": false, "error": fmt.Sprintf("端口 %d 已被占用", port)}, 400
			}
		}
		next.Listeners[index].Port = port
	}
	if egressGroup := stringValue(payload["egress_group"]); egressGroup != "" {
		if !s.hasGroup(egressGroup) {
			return map[string]any{"ok": false, "error": fmt.Sprintf("出口组 %s 不存在", egressGroup)}, 400
		}
		next.Listeners[index].EgressGroup = egressGroup
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
		return map[string]any{"ok": false, "error": fmt.Sprintf("监听 %s 不存在", name)}, 404
	}
	next.Listeners = filtered
	return s.SaveConfig(&next)
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
