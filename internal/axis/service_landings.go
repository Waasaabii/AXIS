package axis

import (
	"fmt"
	"strings"
)

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
	next := s.cloneConfig()
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
	next := s.cloneConfig()
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
	next := s.cloneConfig()
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
