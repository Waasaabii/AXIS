package axis

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"time"
)

const (
	dynamicProxyServiceName         = "dynamic-proxy-service"
	defaultDynamicProxyVersion      = "0.1.0"
	defaultDynamicProxyRateLimitRPM = 120
)

type DynamicProxyCandidate struct {
	ID           string `json:"id"`
	Protocol     string `json:"protocol"`
	ProxyURL     string `json:"proxyUrl"`
	Listener     string `json:"listener"`
	ListenerType string `json:"listenerType"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	EgressGroup  string `json:"egressGroup"`
	CurrentProxy string `json:"currentProxy,omitempty"`
	SearchText   string `json:"-"`
}

type DynamicProxyResult struct {
	OK          bool   `json:"ok"`
	Status      int    `json:"status,omitempty"`
	Code        int    `json:"code,omitempty"`
	Message     string `json:"message,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	ContentType string `json:"-"`
	Format      string `json:"-"`
	Body        any    `json:"-"`
}

type DynamicProxyConfig struct {
	RateLimitPerMinute    int
	TrustForwardedHost    bool
	DefaultResponseFormat string
}

type rateLimitBucket struct {
	Window int64
	Count  int
}

type DynamicProxyService struct {
	service          *Service
	roundRobinCursor int
	rateLimitBuckets map[string]rateLimitBucket
	mu               sync.Mutex
}

func NewDynamicProxyService(service *Service) *DynamicProxyService {
	return &DynamicProxyService{
		service:          service,
		rateLimitBuckets: map[string]rateLimitBucket{},
	}
}

func (d *DynamicProxyService) getConfig() DynamicProxyConfig {
	return DynamicProxyConfig{
		RateLimitPerMinute:    defaultDynamicProxyRateLimitRPM,
		TrustForwardedHost:    false,
		DefaultResponseFormat: "text",
	}
}

func createDynamicRequestID() string {
	return fmt.Sprintf("req_%d_%s", time.Now().UnixMilli(), shortHash(fmt.Sprintf("%d", time.Now().UnixNano())))
}

func normalizeRequestedProtocol(protocol string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(protocol))
	if value == "" {
		return "", nil
	}
	switch value {
	case "http":
		return "http", nil
	case "socks", "socks5":
		return "socks5", nil
	default:
		return "", fmt.Errorf("protocol 仅支持 http 或 socks5")
	}
}

func dynamicListenerProtocols(listenerType string) []string {
	switch strings.TrimSpace(listenerType) {
	case "mixed":
		return []string{"http", "socks5"}
	case "http":
		return []string{"http"}
	default:
		return []string{"socks5"}
	}
}

func normalizeURLHost(host string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]"
	}
	return host
}

func parseHostOnly(rawHost string) string {
	value := strings.TrimSpace(strings.Split(rawHost, ",")[0])
	if value == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	if strings.HasPrefix(value, "[") && strings.Contains(value, "]") {
		return strings.Trim(value, "[]")
	}
	return value
}

func resolvePublicHost(listenerListen string, request *http.Request, trustForwardedHost bool) string {
	listen := strings.TrimSpace(listenerListen)
	switch listen {
	case "", "0.0.0.0", "::", "[::]", ":::":
	default:
		return listen
	}

	if trustForwardedHost {
		if value := parseHostOnly(request.Header.Get("X-Forwarded-Host")); value != "" {
			return value
		}
	}
	if value := parseHostOnly(request.Host); value != "" {
		return value
	}
	return "127.0.0.1"
}

func buildDynamicProxyURL(protocol, host string, port int, user *ListenerUser) string {
	target := fmt.Sprintf("%s://", protocol)
	if user != nil && strings.TrimSpace(user.Username) != "" && user.Password != "" {
		target += neturl.QueryEscape(user.Username) + ":" + neturl.QueryEscape(user.Password) + "@"
	}
	return target + normalizeURLHost(host) + fmt.Sprintf(":%d", port)
}

func buildDynamicSearchText(listener ListenerView) string {
	return strings.ToLower(strings.Join([]string{
		listener.Name,
		listener.EgressGroup,
		listener.TargetName,
		listener.CurrentProxy,
		listener.Type,
		listener.RouteMode,
	}, " "))
}

func matchDynamicFilter(candidate DynamicProxyCandidate, queryValue string) bool {
	filter := strings.ToLower(strings.TrimSpace(queryValue))
	if filter == "" {
		return true
	}
	return strings.Contains(candidate.SearchText, filter)
}

func stableDynamicIndex(seed string, size int) int {
	if size <= 0 {
		return 0
	}
	sum := sha1.Sum([]byte(seed))
	return int(binary.BigEndian.Uint32(sum[:4]) % uint32(size))
}

func preferredDynamicFormat(request *http.Request, defaultFormat string) string {
	switch request.URL.Query().Get("format") {
	case "json":
		return "json"
	case "text":
		return "text"
	}
	accept := request.Header.Get("Accept")
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/plain") {
		return "json"
	}
	if defaultFormat == "json" {
		return "json"
	}
	return "text"
}

func dynamicClientID(request *http.Request) string {
	if request == nil {
		return "unknown"
	}
	if host, _, err := net.SplitHostPort(request.RemoteAddr); err == nil && host != "" {
		return host
	}
	if strings.TrimSpace(request.RemoteAddr) != "" {
		return request.RemoteAddr
	}
	return "unknown"
}

func (d *DynamicProxyService) enforceRateLimit(request *http.Request) *DynamicProxyResult {
	config := d.getConfig()
	if config.RateLimitPerMinute <= 0 {
		return nil
	}

	clientID := dynamicClientID(request)
	currentWindow := time.Now().Unix() / 60

	d.mu.Lock()
	defer d.mu.Unlock()

	bucket, ok := d.rateLimitBuckets[clientID]
	if !ok || bucket.Window != currentWindow {
		d.rateLimitBuckets[clientID] = rateLimitBucket{Window: currentWindow, Count: 1}
		return nil
	}
	if bucket.Count >= config.RateLimitPerMinute {
		return &DynamicProxyResult{
			OK:        false,
			Status:    http.StatusTooManyRequests,
			Code:      42901,
			Message:   "rate limit exceeded",
			RequestID: createDynamicRequestID(),
		}
	}
	bucket.Count++
	d.rateLimitBuckets[clientID] = bucket
	return nil
}

func (d *DynamicProxyService) buildCandidatePool(request *http.Request) []DynamicProxyCandidate {
	listeners := d.service.GetListeners()
	config := d.getConfig()

	candidates := []DynamicProxyCandidate{}
	for _, listener := range listeners {
		if !listener.Enabled || listener.Status != "configured" {
			continue
		}
		host := resolvePublicHost(listener.Listen, request, config.TrustForwardedHost)
		var primaryUser *ListenerUser
		if len(listener.Users) > 0 {
			primaryUser = &listener.Users[0]
		}
		searchText := buildDynamicSearchText(listener)
		for _, protocol := range dynamicListenerProtocols(listener.Type) {
			candidates = append(candidates, DynamicProxyCandidate{
				ID:           fmt.Sprintf("%s:%s", listener.Name, protocol),
				Protocol:     protocol,
				ProxyURL:     buildDynamicProxyURL(protocol, host, listener.Port, primaryUser),
				Listener:     listener.Name,
				ListenerType: listener.Type,
				Host:         host,
				Port:         listener.Port,
				EgressGroup:  listener.EgressGroup,
				CurrentProxy: listener.CurrentProxy,
				SearchText:   searchText,
			})
		}
	}
	return candidates
}

func (d *DynamicProxyService) selectCandidate(candidates []DynamicProxyCandidate, session string) *DynamicProxyCandidate {
	if len(candidates) == 0 {
		return nil
	}
	if strings.TrimSpace(session) != "" {
		candidate := candidates[stableDynamicIndex(session, len(candidates))]
		return &candidate
	}

	d.mu.Lock()
	index := d.roundRobinCursor % len(candidates)
	d.roundRobinCursor = (d.roundRobinCursor + 1) % int(^uint(0)>>1)
	d.mu.Unlock()

	candidate := candidates[index]
	return &candidate
}

func (d *DynamicProxyService) GetNextProxy(request *http.Request) DynamicProxyResult {
	requestID := createDynamicRequestID()
	if rateLimitError := d.enforceRateLimit(request); rateLimitError != nil {
		return *rateLimitError
	}

	protocol, err := normalizeRequestedProtocol(request.URL.Query().Get("protocol"))
	if err != nil {
		return DynamicProxyResult{
			OK:        false,
			Status:    http.StatusBadRequest,
			Code:      40001,
			Message:   err.Error(),
			RequestID: requestID,
		}
	}

	allCandidates := d.buildCandidatePool(request)
	filteredCandidates := make([]DynamicProxyCandidate, 0, len(allCandidates))
	for _, candidate := range allCandidates {
		if protocol != "" && candidate.Protocol != protocol {
			continue
		}
		if !matchDynamicFilter(candidate, request.URL.Query().Get("region")) {
			continue
		}
		if !matchDynamicFilter(candidate, request.URL.Query().Get("city")) {
			continue
		}
		if !matchDynamicFilter(candidate, request.URL.Query().Get("tag")) {
			continue
		}
		filteredCandidates = append(filteredCandidates, candidate)
	}

	if len(allCandidates) == 0 {
		return DynamicProxyResult{
			OK:        false,
			Status:    http.StatusServiceUnavailable,
			Code:      50301,
			Message:   "no proxy available",
			RequestID: requestID,
		}
	}
	if len(filteredCandidates) == 0 {
		return DynamicProxyResult{
			OK:        false,
			Status:    http.StatusConflict,
			Code:      40901,
			Message:   "no proxy available for current filters",
			RequestID: requestID,
		}
	}

	session := request.URL.Query().Get("session")
	candidate := d.selectCandidate(filteredCandidates, session)
	if candidate == nil {
		return DynamicProxyResult{
			OK:        false,
			Status:    http.StatusServiceUnavailable,
			Code:      50301,
			Message:   "no proxy available",
			RequestID: requestID,
		}
	}

	format := preferredDynamicFormat(request, d.getConfig().DefaultResponseFormat)
	result := DynamicProxyResult{
		OK:          true,
		Status:      http.StatusOK,
		RequestID:   requestID,
		Format:      format,
		ContentType: ternaryString(format == "json", "application/json; charset=utf-8", "text/plain; charset=utf-8"),
	}
	if format == "json" {
		result.Body = map[string]any{
			"code":    0,
			"message": "ok",
			"data": map[string]any{
				"proxy":         candidate.ProxyURL,
				"provider":      candidate.Listener,
				"protocol":      candidate.Protocol,
				"region":        emptyStringToNil(request.URL.Query().Get("region")),
				"city":          emptyStringToNil(request.URL.Query().Get("city")),
				"session_id":    emptyStringToNil(session),
				"listener":      candidate.Listener,
				"egress_group":  candidate.EgressGroup,
				"current_proxy": emptyStringToNil(candidate.CurrentProxy),
			},
		}
	} else {
		result.Body = candidate.ProxyURL
	}
	return result
}

func (d *DynamicProxyService) GetHealth(request *http.Request) map[string]any {
	listeners := d.service.GetListeners()
	total := 0
	available := 0
	for _, listener := range listeners {
		if !listener.Enabled {
			continue
		}
		protocolCount := len(dynamicListenerProtocols(listener.Type))
		total += protocolCount
		if listener.Status == "configured" {
			available += protocolCount
		}
	}
	return map[string]any{
		"status":        "ok",
		"service":       dynamicProxyServiceName,
		"version":       defaultDynamicProxyVersion,
		"time":          nowISO(),
		"auth_required": false,
		"pool": map[string]any{
			"total":     total,
			"available": available,
			"cooling":   0,
			"disabled":  max(total-available, 0),
		},
		"request_host": resolvePublicHost("0.0.0.0", request, d.getConfig().TrustForwardedHost),
	}
}

func emptyStringToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
