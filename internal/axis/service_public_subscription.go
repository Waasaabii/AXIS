package axis

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type PublishedSubscription struct {
	Name    string
	Content string
}

func (s *Service) BuildPublishedSubscription(token string) (PublishedSubscription, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return PublishedSubscription{}, errors.New("订阅令牌不能为空")
	}
	for _, publication := range s.config.Publications {
		if !publication.Enabled || publication.Type != "subscription" {
			continue
		}
		if strings.TrimSpace(publication.Auth.Token) != token {
			continue
		}
		content, err := s.renderPublicationSubscription(publication)
		if err != nil {
			return PublishedSubscription{}, err
		}
		return PublishedSubscription{Name: publication.Name, Content: content}, nil
	}
	return PublishedSubscription{}, errors.New("订阅不存在或令牌不正确")
}

func (s *Service) renderPublicationSubscription(publication PublicationConfig) (string, error) {
	route, ok := s.findRoute(publication.Route)
	if !ok {
		return "", fmt.Errorf("发布绑定的线路 %s 不存在", publication.Route)
	}
	items := []map[string]any{}
	if route.Entry.Source != "" {
		items = append(items, s.renderPublishedSource(route.Entry.Source)...)
	}
	if route.Landing.Source != "" {
		items = append(items, s.renderPublishedSource(route.Landing.Source)...)
	}
	if len(items) == 0 {
		return "", errors.New("发布线路没有可输出的节点")
	}
	proxyNames := make([]string, 0, len(items))
	for _, item := range items {
		if name := strings.TrimSpace(fmt.Sprint(item["name"])); name != "" {
			proxyNames = append(proxyNames, name)
		}
	}
	document := map[string]any{
		"proxies": items,
		"proxy-groups": []map[string]any{{
			"name":     firstNonEmpty(publication.Name, route.Name),
			"type":     "url-test",
			"proxies":  proxyNames,
			"url":      firstNonEmpty(route.HealthCheck.URL, "https://www.gstatic.com/generate_204"),
			"interval": max(route.HealthCheck.Interval, 300),
		}},
		"rules": []string{fmt.Sprintf("MATCH,%s", firstNonEmpty(publication.Name, route.Name))},
	}
	raw, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *Service) findRoute(name string) (RouteConfig, bool) {
	for _, route := range s.config.Routes {
		if route.Name == name && route.Enabled {
			return route, true
		}
	}
	return RouteConfig{}, false
}

func (s *Service) renderPublishedSource(sourceName string) []map[string]any {
	for _, source := range s.config.NodeSources {
		if source.Name != sourceName || !source.Enabled {
			continue
		}
		switch source.Type {
		case "proxy":
			if item := renderPublishedEndpoint(source.Name, source.Protocol, source.Endpoint); item != nil {
				return []map[string]any{item}
			}
		case "local_node":
			endpoint := NodeSourceEndpoint{Server: source.LocalNode.ExternalHost, Port: source.LocalNode.ExternalPort, Username: firstLocalUser(source.LocalNode.Users).Username, Password: firstLocalUser(source.LocalNode.Users).Password, TLS: source.LocalNode.TLS, SNI: source.LocalNode.SNI, SkipCertVerify: source.LocalNode.SkipCertVerify}
			if endpoint.Port == 0 {
				endpoint.Port = defaultLocalNodeExternalPort(source.Protocol, source.LocalNode.Port)
			}
			if item := renderPublishedEndpoint(source.Name, source.Protocol, endpoint); item != nil {
				return []map[string]any{item}
			}
		case "subscription", "axis":
			return s.renderSubscriptionSnapshot(source)
		}
	}
	return nil
}

func (s *Service) renderSubscriptionSnapshot(source NodeSource) []map[string]any {
	url := source.Subscription.URL
	if source.Type == "axis" {
		url = source.Axis.URL
	}
	if strings.TrimSpace(url) == "" {
		return nil
	}
	record, ok, _, _, err := FetchProviderSnapshot(Subscription{Name: source.Name, Type: firstNonEmpty(source.Protocol, "mihomo-http"), URL: url, Interval: source.Subscription.Interval, HealthCheckURL: source.Subscription.HealthCheckURL, HealthCheckInterval: source.Subscription.HealthCheckInterval, Headers: source.Subscription.Headers, Via: source.Subscription.Via})
	if err != nil || !ok || record == nil {
		return nil
	}
	items := make([]map[string]any, 0, len(record.Nodes))
	for _, node := range record.Nodes {
		if item := renderPublishedNodeInfo(node); item != nil {
			items = append(items, item)
		}
	}
	return items
}

func renderPublishedEndpoint(name, protocol string, endpoint NodeSourceEndpoint) map[string]any {
	protocol = normalizeManualProviderType(protocol)
	if endpoint.Server == "" || endpoint.Port <= 0 {
		return nil
	}
	item := map[string]any{"name": name, "type": protocol, "server": endpoint.Server, "port": endpoint.Port}
	if endpoint.Username != "" {
		item["username"] = endpoint.Username
	}
	if endpoint.Password != "" {
		item["password"] = endpoint.Password
	}
	if endpoint.TLS {
		item["tls"] = true
	}
	if endpoint.SNI != "" {
		item["sni"] = endpoint.SNI
	}
	if endpoint.SkipCertVerify {
		item["skip-cert-verify"] = true
	}
	return item
}

func renderPublishedNodeInfo(node NodeInfo) map[string]any {
	protocol := strings.TrimSpace(node.Type)
	if protocol == "" || node.Server == "" || node.Port <= 0 {
		return nil
	}
	item := map[string]any{"name": node.Name, "type": protocol, "server": node.Server, "port": node.Port}
	if node.TLS != "" && node.TLS != "<nil>" {
		if node.TLS == "tls" || node.TLS == "true" {
			item["tls"] = true
		}
	}
	return item
}
