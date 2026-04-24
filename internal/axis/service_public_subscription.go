package axis

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type PublishedSubscription struct {
	Name        string
	Content     string
	ContentType string
	Extension   string
}

func (s *Service) BuildPublishedSubscription(token, format string) (PublishedSubscription, error) {
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
		content, contentType, extension, err := s.renderPublicationSubscription(publication, format)
		if err != nil {
			return PublishedSubscription{}, err
		}
		return PublishedSubscription{Name: publication.Name, Content: content, ContentType: contentType, Extension: extension}, nil
	}
	return PublishedSubscription{}, errors.New("订阅不存在或令牌不正确")
}

func (s *Service) renderPublicationSubscription(publication PublicationConfig, format string) (string, string, string, error) {
	route, ok := s.findRoute(publication.Route)
	if !ok {
		return "", "", "", fmt.Errorf("发布绑定的线路 %s 不存在", publication.Route)
	}
	items := []map[string]any{}
	if route.Entry.Source != "" {
		items = append(items, s.renderPublishedSource(route.Entry.Source)...)
	}
	if route.Landing.Source != "" {
		items = append(items, s.renderPublishedSource(route.Landing.Source)...)
	}
	if len(items) == 0 {
		return "", "", "", errors.New("发布线路没有可输出的节点")
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
	if isURIPublicationFormat(format) {
		content := renderPublicationURIList(items)
		if strings.TrimSpace(content) == "" {
			return "", "", "", errors.New("发布线路没有可输出的分享链接")
		}
		return content, "text/plain; charset=utf-8", "txt", nil
	}
	raw, err := yaml.Marshal(document)
	if err != nil {
		return "", "", "", err
	}
	return string(raw), "text/yaml; charset=utf-8", "yaml", nil
}

func isURIPublicationFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "uri", "url", "share", "shadowrocket":
		return true
	default:
		return false
	}
}

func renderPublicationURIList(items []map[string]any) string {
	links := make([]string, 0, len(items))
	for _, item := range items {
		if link := renderPublicationURI(item); link != "" {
			links = append(links, link)
		}
	}
	return strings.Join(links, "\n")
}

func renderPublicationURI(item map[string]any) string {
	protocol := strings.ToLower(strings.TrimSpace(fmt.Sprint(item["type"])))
	server := strings.TrimSpace(fmt.Sprint(item["server"]))
	password := strings.TrimSpace(fmt.Sprint(firstNonNil(item["password"], item["uuid"])))
	port, _ := strconv.Atoi(fmt.Sprint(item["port"]))
	if protocol == "" || server == "" || password == "" || port <= 0 {
		return ""
	}
	name := url.QueryEscape(strings.TrimSpace(fmt.Sprint(item["name"])))
	sni := strings.TrimSpace(fmt.Sprint(item["sni"]))
	query := url.Values{}
	copyStringQuery(query, item, "sni", "sni")
	copyStringQuery(query, item, "peer", "peer")
	copyStringQuery(query, item, "host", "host")
	copyStringQuery(query, item, "path", "path")
	copyStringQuery(query, item, "network", "type")
	copyStringQuery(query, item, "alpn", "alpn")
	copyStringQuery(query, item, "obfs", "obfs")
	if skip, ok := item["skip-cert-verify"].(bool); ok && skip {
		query.Set("insecure", "1")
	}
	if allow, ok := item["allowInsecure"].(bool); ok {
		if allow {
			query.Set("allowInsecure", "1")
		} else if protocol == "trojan" {
			query.Set("allowInsecure", "0")
		}
	}
	if extra, ok := item["query"].(map[string]any); ok {
		for key, value := range extra {
			if strings.TrimSpace(fmt.Sprint(value)) != "" && query.Get(key) == "" {
				query.Set(key, fmt.Sprint(value))
			}
		}
	}
	switch protocol {
	case "trojan":
		if query.Get("peer") == "" && sni != "" {
			query.Set("peer", sni)
		}
		if query.Get("allowInsecure") == "" && query.Get("insecure") == "" {
			query.Set("allowInsecure", "0")
		}
		return fmt.Sprintf("trojan://%s@%s:%d?%s#%s", url.QueryEscape(password), server, port, query.Encode(), name)
	case "hysteria2":
		if query.Get("insecure") == "" {
			query.Set("insecure", "0")
		}
		return fmt.Sprintf("hysteria2://%s@%s:%d/?%s#%s", url.QueryEscape(password), server, port, query.Encode(), name)
	default:
		return ""
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if strings.TrimSpace(fmt.Sprint(value)) != "" && strings.TrimSpace(fmt.Sprint(value)) != "<nil>" {
			return value
		}
	}
	return ""
}

func copyStringQuery(query url.Values, item map[string]any, sourceKey, queryKey string) {
	value := strings.TrimSpace(fmt.Sprint(item[sourceKey]))
	if value == "" || value == "<nil>" {
		return
	}
	query.Set(queryKey, value)
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
	if protocol == "trojan" || protocol == "hysteria2" {
		item["udp"] = true
	}
	if endpoint.TLS && protocol != "hysteria2" {
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
	setIfNotEmpty(item, "username", node.Username)
	setIfNotEmpty(item, "password", firstNonEmpty(node.Password, node.UUID))
	setIfNotEmpty(item, "uuid", node.UUID)
	setIfNotEmpty(item, "network", node.Network)
	setIfNotEmpty(item, "sni", node.SNI)
	setIfNotEmpty(item, "peer", node.Peer)
	setIfNotEmpty(item, "host", node.Host)
	setIfNotEmpty(item, "path", node.Path)
	setIfNotEmpty(item, "obfs", node.Obfs)
	if len(node.ALPN) > 0 {
		item["alpn"] = strings.Join(node.ALPN, ",")
	}
	if node.TLS != "" && node.TLS != "<nil>" {
		if node.TLS == "tls" || node.TLS == "true" {
			item["tls"] = true
		} else {
			item["tls"] = node.TLS
		}
	}
	if node.SkipCertVerify {
		item["skip-cert-verify"] = true
	}
	item["allowInsecure"] = node.AllowInsecure
	if len(node.Query) > 0 {
		item["query"] = node.Query
	}
	if len(node.Extra) > 0 {
		item["extra"] = node.Extra
	}
	return item
}

func setIfNotEmpty(item map[string]any, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" && value != "<nil>" {
		item[key] = value
	}
}
