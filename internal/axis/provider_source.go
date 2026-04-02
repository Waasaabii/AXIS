package axis

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var manualProviderTypes = map[string]struct{}{
	"http":   {},
	"socks":  {},
	"socks5": {},
}

var providerFileNamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func isManualProviderType(providerType string) bool {
	_, ok := manualProviderTypes[strings.TrimSpace(strings.ToLower(providerType))]
	return ok
}

func normalizeManualProviderType(providerType string) string {
	if strings.EqualFold(strings.TrimSpace(providerType), "http") {
		return "http"
	}
	return "socks5"
}

func buildManualProviderEndpoint(subscription Subscription) string {
	protocol := strings.TrimSpace(strings.ToLower(subscription.Type))
	if protocol == "" {
		protocol = "socks5"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, strings.TrimSpace(subscription.Server), subscription.Port)
}

func buildManualProviderName(providerType, server string, port int) string {
	sanitize := func(value, fallback string) string {
		normalized := providerFileNamePattern.ReplaceAllString(strings.TrimSpace(value), "-")
		normalized = strings.Trim(normalized, "-")
		if normalized == "" {
			return fallback
		}
		return normalized
	}

	return strings.Join([]string{
		sanitize(providerType, "node"),
		sanitize(server, "host"),
		sanitize(strconv.Itoa(port), "0"),
	}, "-")
}

func providerFileName(name string) string {
	normalized := providerFileNamePattern.ReplaceAllString(strings.TrimSpace(name), "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "provider"
	}
	return normalized + ".yaml"
}

func parseManualProviderInput(input, fallbackType string) (Subscription, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return Subscription{}, nil
	}

	match := regexp.MustCompile(`^([a-zA-Z0-9+.-]+)://(.+)$`).FindStringSubmatch(text)
	providerType := strings.TrimSpace(strings.ToLower(fallbackType))
	remainder := text
	if len(match) == 3 {
		providerType = strings.TrimSpace(strings.ToLower(match[1]))
		remainder = strings.TrimSpace(match[2])
	}

	if !isManualProviderType(providerType) {
		return Subscription{}, fmt.Errorf("导入串仅支持 HTTP、SOCKS、SOCKS5")
	}

	if strings.Contains(remainder, "@") {
		parsedURL, err := url.Parse(fmt.Sprintf("%s://%s", providerType, remainder))
		if err != nil {
			return Subscription{}, fmt.Errorf("导入串格式无效")
		}
		port, err := strconv.Atoi(parsedURL.Port())
		if err != nil || port <= 0 || strings.TrimSpace(parsedURL.Hostname()) == "" {
			return Subscription{}, fmt.Errorf("导入串中的服务器或端口无效")
		}
		return Subscription{
			Type:     providerType,
			Server:   strings.TrimSpace(parsedURL.Hostname()),
			Port:     port,
			Username: parsedURL.User.Username(),
			Password: func() string { password, _ := parsedURL.User.Password(); return password }(),
		}, nil
	}

	parts := strings.Split(remainder, ":")
	if len(parts) < 2 {
		return Subscription{}, fmt.Errorf("导入串格式无效，至少需要协议、IP 和端口")
	}

	port, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || port <= 0 || strings.TrimSpace(parts[0]) == "" {
		return Subscription{}, fmt.Errorf("导入串中的服务器或端口无效")
	}

	subscription := Subscription{
		Type:   providerType,
		Server: strings.TrimSpace(parts[0]),
		Port:   port,
	}
	if len(parts) >= 3 {
		subscription.Username = strings.TrimSpace(parts[2])
	}
	if len(parts) >= 4 {
		subscription.Password = strings.Join(parts[3:], ":")
	}
	return subscription, nil
}

func buildManualProviderSnapshot(subscription Subscription) *ProviderRecord {
	node := NodeInfo{
		ID:      createNodeID(subscription.Name, buildManualProviderEndpoint(subscription)),
		Name:    strings.TrimSpace(subscription.Name),
		Type:    normalizeManualProviderType(subscription.Type),
		Server:  strings.TrimSpace(subscription.Server),
		Port:    subscription.Port,
		Source:  "manual",
		Network: "",
		TLS:     "",
	}

	return &ProviderRecord{
		Provider:    subscription.Name,
		URLMasked:   buildManualProviderEndpoint(subscription),
		RefreshedAt: nowISO(),
		NodeCount:   1,
		Nodes:       []NodeInfo{node},
	}
}

func buildManualProviderFileContent(subscription Subscription) ([]byte, error) {
	proxy := map[string]any{
		"name":   strings.TrimSpace(subscription.Name),
		"type":   normalizeManualProviderType(subscription.Type),
		"server": strings.TrimSpace(subscription.Server),
		"port":   subscription.Port,
	}
	if strings.TrimSpace(subscription.Username) != "" {
		proxy["username"] = strings.TrimSpace(subscription.Username)
	}
	if subscription.Password != "" {
		proxy["password"] = subscription.Password
	}
	return yaml.Marshal(map[string]any{
		"proxies": []map[string]any{proxy},
	})
}
