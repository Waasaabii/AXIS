package axis

import (
	"fmt"
	"strings"
)

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

	next := s.cloneConfig()
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
	next := s.cloneConfig()
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

	next := s.cloneConfig()
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
