package axis

import (
	"fmt"
	"strings"
)

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

	next := s.cloneConfig()
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

func (s *Service) UpdateListener(name string, payload map[string]any) (map[string]any, int) {
	next := s.cloneConfig()
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
	next := s.cloneConfig()
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
