package axis

import (
	"fmt"
	"strings"
)

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
		sourceKind := "subscription"
		endpoint := firstNonEmpty(record.URLMasked, subscription.URL)
		if isManualProviderType(subscription.Type) {
			sourceKind = "manual-node"
			endpoint = firstNonEmpty(record.URLMasked, buildManualProviderEndpoint(subscription))
		}
		items = append(items, map[string]any{
			"name":           subscription.Name,
			"type":           firstNonEmpty(subscription.Type, "mihomo-http"),
			"urlMasked":      endpoint,
			"endpoint":       endpoint,
			"interval":       max(subscription.Interval, 3600),
			"enabled":        subscription.Enabled,
			"refreshedAt":    record.RefreshedAt,
			"nodeCount":      record.NodeCount,
			"lastError":      record.LastError,
			"nodes":          record.Nodes,
			"sourceKind":     sourceKind,
			"manual":         isManualProviderType(subscription.Type),
			"hasCredentials": strings.TrimSpace(subscription.Username) != "" || subscription.Password != "",
		})
	}
	return items
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

func (s *Service) RefreshProvider(providerName string) map[string]any {
	subscription := s.findSubscription(providerName)
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
	runtimePayload := map[string]any{}
	if isManualProviderType(subscription.Type) {
		runtimePayload["ok"] = false
		runtimePayload["deferred"] = true
		runtimePayload["message"] = "手动节点无需远程刷新代理核心订阅。"
	} else {
		runtimeRefresh, runtimeErr := s.controller.RefreshProvider(providerName)
		if runtimeErr != nil {
			runtimePayload["ok"] = false
			runtimePayload["error"] = runtimeErr.Error()
		} else {
			runtimePayload["ok"] = runtimeRefresh.OK
			runtimePayload["status"] = runtimeRefresh.Status
			runtimePayload["payload"] = runtimeRefresh.Payload
		}
	}
	s.pushEvent(ternaryString(ok, "info", "warn"), "provider", fmt.Sprintf("%s 刷新完成，节点数 %d", providerName, record.NodeCount))
	_ = s.persistState()
	return map[string]any{"ok": ok, "provider": s.state.Providers[providerName], "runtime": runtimePayload}
}

func extractHealthcheckDelay(payload any) int {
	switch value := payload.(type) {
	case map[string]any:
		for _, key := range []string{"delay", "meanDelay"} {
			switch delay := value[key].(type) {
			case float64:
				return int(delay)
			case int:
				return delay
			}
		}
	case []any:
		best := 0
		for _, item := range value {
			delay := extractHealthcheckDelay(item)
			if delay > 0 && (best == 0 || delay < best) {
				best = delay
			}
		}
		return best
	}
	return 0
}
