package axis

import (
	"fmt"
	"strings"
)

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

	next := s.cloneConfig()
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

func (s *Service) RemoveSubscription(name string) (map[string]any, int) {
	for _, route := range s.config.TransitRoutes {
		if route.UpstreamProvider == name {
			return map[string]any{"ok": false, "error": fmt.Sprintf("订阅 %s 仍被中转线路 %s 使用，请先解除绑定", name, route.Name)}, 400
		}
	}

	found := false
	next := s.cloneConfig()
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
	next := s.cloneConfig()
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
	next := s.cloneConfig()
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
