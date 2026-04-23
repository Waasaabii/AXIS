package axis

import (
	"encoding/json"
	"fmt"
)

func decodeIntoConfig(payload map[string]any) (*Config, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func DecodeConfigPayload(payload map[string]any) (*Config, error) {
	if payload == nil {
		return nil, fmt.Errorf("config payload 不能为空")
	}

	configPayload := payload
	if nested, ok := payload["config"].(map[string]any); ok {
		configPayload = nested
	}
	if len(configPayload) == 0 {
		return nil, fmt.Errorf("config 不能为空")
	}
	return decodeIntoConfig(configPayload)
}

func stringValue(input any) string {
	if value, ok := input.(string); ok {
		return value
	}
	return ""
}

func intValue(input any, fallback int) int {
	switch value := input.(type) {
	case float64:
		return int(value)
	case int:
		return value
	}
	return fallback
}

func boolValue(input any, fallback bool) bool {
	if value, ok := input.(bool); ok {
		return value
	}
	return fallback
}

func stringArrayValue(input any) []string {
	items, ok := input.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			values = append(values, value)
		}
	}
	return values
}

func parseUsers(input any) []ListenerUser {
	items, ok := input.([]any)
	if !ok {
		return nil
	}
	users := []ListenerUser{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		users = append(users, ListenerUser{Username: stringValue(entry["username"]), Password: stringValue(entry["password"])})
	}
	return users
}
