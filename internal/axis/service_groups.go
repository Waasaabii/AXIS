package axis

import (
	"fmt"
	"strings"
)

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
	next := s.cloneConfig()
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

func (s *Service) UpdateEgressGroup(name string, payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
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
	next := s.cloneConfig()
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
