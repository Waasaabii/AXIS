package axis

func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func jsonBody(schema map[string]any) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": schema,
			},
		},
	}
}

func jsonResponse(description string, schema map[string]any) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": schema,
			},
		},
	}
}

func noContentResponse(description string) map[string]any {
	return map[string]any{"description": description}
}

func pathParam(name, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "path",
		"required":    true,
		"description": description,
		"schema": map[string]any{
			"type": "string",
		},
	}
}

func mutationErrorResponses() map[string]any {
	return map[string]any{
		"400": jsonResponse("请求参数错误", schemaRef("ErrorResponse")),
		"401": jsonResponse("未登录或会话已过期", schemaRef("ErrorResponse")),
		"404": jsonResponse("目标不存在", schemaRef("ErrorResponse")),
		"500": jsonResponse("服务端错误", schemaRef("ErrorResponse")),
	}
}

func BuildOpenAPISpec() map[string]any {
	schemas := map[string]any{
		"ErrorResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":    map[string]any{"type": "boolean"},
				"error": map[string]any{"type": "string"},
			},
			"required": []string{"ok", "error"},
		},
		"SimpleOkResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok": map[string]any{"type": "boolean"},
			},
			"required": []string{"ok"},
		},
		"UserIdentity": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"username": map[string]any{"type": "string"},
			},
			"required": []string{"username"},
		},
		"SessionStatusResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"authenticated":         map[string]any{"type": "boolean"},
				"requiresPasswordReset": map[string]any{"type": "boolean"},
				"setupRequired":         map[string]any{"type": "boolean"},
				"user":                  schemaRef("UserIdentity"),
			},
			"required": []string{"authenticated"},
		},
		"LoginRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"username": map[string]any{"type": "string"},
				"password": map[string]any{"type": "string"},
			},
			"required": []string{"username", "password"},
		},
		"LoginResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":                    map[string]any{"type": "boolean"},
				"requiresPasswordReset": map[string]any{"type": "boolean"},
				"setupRequired":         map[string]any{"type": "boolean"},
				"user":                  schemaRef("UserIdentity"),
			},
			"required": []string{"ok", "requiresPasswordReset", "setupRequired", "user"},
		},
		"UpdatePasswordRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"password": map[string]any{"type": "string"},
			},
			"required": []string{"password"},
		},
		"RuntimeApplyResult": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":       map[string]any{"type": "boolean"},
				"deferred": map[string]any{"type": "boolean"},
				"status":   map[string]any{"type": "integer"},
				"message":  map[string]any{"type": "string"},
			},
			"required": []string{"message"},
		},
		"ServerConfig": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"host":            map[string]any{"type": "string"},
				"port":            map[string]any{"type": "integer"},
				"startup_refresh": map[string]any{"type": "boolean"},
			},
			"required": []string{"host", "port", "startup_refresh"},
		},
		"AdminConfig": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"username":                map[string]any{"type": "string"},
				"password":                map[string]any{"type": "string"},
				"password_hash":           map[string]any{"type": "string"},
				"requires_password_reset": map[string]any{"type": "boolean"},
				"session_secret":          map[string]any{"type": "string"},
				"session_ttl_hours":       map[string]any{"type": "integer"},
			},
			"required": []string{"username", "password", "password_hash", "session_secret", "session_ttl_hours"},
		},
		"RuntimeConfig": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workdir":             map[string]any{"type": "string"},
				"mihomo_binary":       map[string]any{"type": "string"},
				"external_controller": map[string]any{"type": "string"},
				"external_secret":     map[string]any{"type": "string"},
				"render_only":         map[string]any{"type": "boolean"},
			},
			"required": []string{"workdir", "mihomo_binary", "external_controller", "external_secret", "render_only"},
		},
		"Subscription": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":                  map[string]any{"type": "string"},
				"type":                  map[string]any{"type": "string"},
				"url":                   map[string]any{"type": "string"},
				"interval":              map[string]any{"type": "integer"},
				"enabled":               map[string]any{"type": "boolean"},
				"health_check_url":      map[string]any{"type": "string"},
				"health_check_interval": map[string]any{"type": "integer"},
				"headers": map[string]any{
					"type":                 "object",
					"additionalProperties": map[string]any{"type": "string"},
				},
				"via": map[string]any{"type": "string"},
			},
			"required": []string{"name", "type", "url", "interval", "enabled"},
		},
		"EgressGroup": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":             map[string]any{"type": "string"},
				"provider":         map[string]any{"type": "string"},
				"mode":             map[string]any{"type": "string"},
				"filter":           map[string]any{"type": "string"},
				"exclude_filter":   map[string]any{"type": "string"},
				"strategy":         map[string]any{"type": "string"},
				"health_check_url": map[string]any{"type": "string"},
				"interval":         map[string]any{"type": "integer"},
				"fallback":         map[string]any{"type": "string"},
			},
			"required": []string{"name", "provider", "mode", "filter", "exclude_filter"},
		},
		"ListenerUser": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"username": map[string]any{"type": "string"},
				"password": map[string]any{"type": "string"},
			},
			"required": []string{"username", "password"},
		},
		"Listener": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":         map[string]any{"type": "string"},
				"type":         map[string]any{"type": "string"},
				"listen":       map[string]any{"type": "string"},
				"port":         map[string]any{"type": "integer"},
				"udp":          map[string]any{"type": "boolean"},
				"enabled":      map[string]any{"type": "boolean"},
				"users":        map[string]any{"type": "array", "items": schemaRef("ListenerUser")},
				"egress_group": map[string]any{"type": "string"},
			},
			"required": []string{"name", "type", "listen", "port", "udp", "enabled", "users", "egress_group"},
		},
		"Config": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"server":        schemaRef("ServerConfig"),
				"admin":         schemaRef("AdminConfig"),
				"runtime":       schemaRef("RuntimeConfig"),
				"subscriptions": map[string]any{"type": "array", "items": schemaRef("Subscription")},
				"egress_groups": map[string]any{"type": "array", "items": schemaRef("EgressGroup")},
				"listeners":     map[string]any{"type": "array", "items": schemaRef("Listener")},
			},
			"required": []string{"server", "admin", "runtime", "subscriptions", "egress_groups", "listeners"},
		},
		"ConfigEnvelope": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":   map[string]any{"type": "string"},
				"config": schemaRef("Config"),
			},
			"required": []string{"path", "config"},
		},
		"SaveConfigRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"config": schemaRef("Config"),
			},
			"required": []string{"config"},
		},
		"SaveConfigResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":           map[string]any{"type": "boolean"},
				"path":         map[string]any{"type": "string"},
				"config":       schemaRef("Config"),
				"runtimeApply": schemaRef("RuntimeApplyResult"),
			},
			"required": []string{"ok", "path", "config", "runtimeApply"},
		},
		"NodeInfo": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":      map[string]any{"type": "string"},
				"name":    map[string]any{"type": "string"},
				"type":    map[string]any{"type": "string"},
				"server":  map[string]any{"type": "string"},
				"port":    map[string]any{"type": "integer"},
				"network": map[string]any{"type": "string"},
				"tls":     map[string]any{"type": "string"},
				"source":  map[string]any{"type": "string"},
			},
			"required": []string{"id", "name", "type", "server", "port"},
		},
		"ProviderRecord": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"provider":    map[string]any{"type": "string"},
				"urlMasked":   map[string]any{"type": "string"},
				"refreshedAt": map[string]any{"type": "string"},
				"nodeCount":   map[string]any{"type": "integer"},
				"nodes":       map[string]any{"type": "array", "items": schemaRef("NodeInfo")},
				"lastError":   map[string]any{"type": "string"},
			},
			"required": []string{"provider", "nodeCount", "nodes"},
		},
		"ProviderListItem": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string"},
				"type":        map[string]any{"type": "string"},
				"urlMasked":   map[string]any{"type": "string"},
				"interval":    map[string]any{"type": "integer"},
				"enabled":     map[string]any{"type": "boolean"},
				"refreshedAt": map[string]any{"type": "string"},
				"nodeCount":   map[string]any{"type": "integer"},
				"lastError":   map[string]any{"type": "string"},
				"nodes":       map[string]any{"type": "array", "items": schemaRef("NodeInfo")},
			},
			"required": []string{"name", "type", "urlMasked", "interval", "enabled", "nodeCount", "nodes"},
		},
		"GroupCandidate": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":     map[string]any{"type": "string"},
				"name":   map[string]any{"type": "string"},
				"type":   map[string]any{"type": "string"},
				"server": map[string]any{"type": "string"},
				"port":   map[string]any{"type": "integer"},
			},
			"required": []string{"id", "name", "type", "server", "port"},
		},
		"GroupView": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":              map[string]any{"type": "string"},
				"mode":              map[string]any{"type": "string"},
				"provider":          map[string]any{"type": "string"},
				"filter":            map[string]any{"type": "string"},
				"candidateCount":    map[string]any{"type": "integer"},
				"current":           map[string]any{"type": "string"},
				"candidates":        map[string]any{"type": "array", "items": schemaRef("GroupCandidate")},
				"lastHealthcheckAt": map[string]any{"type": "string"},
				"providerMissing":   map[string]any{"type": "boolean"},
				"providerDisabled":  map[string]any{"type": "boolean"},
			},
			"required": []string{"name", "mode", "provider", "filter", "candidateCount", "candidates", "providerMissing", "providerDisabled"},
		},
		"ListenerView": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":             map[string]any{"type": "string"},
				"type":             map[string]any{"type": "string"},
				"listen":           map[string]any{"type": "string"},
				"port":             map[string]any{"type": "integer"},
				"udp":              map[string]any{"type": "boolean"},
				"users":            map[string]any{"type": "array", "items": schemaRef("ListenerUser")},
				"userCount":        map[string]any{"type": "integer"},
				"egressGroup":      map[string]any{"type": "string"},
				"currentProxy":     map[string]any{"type": "string"},
				"status":           map[string]any{"type": "string"},
				"groupMissing":     map[string]any{"type": "boolean"},
				"providerMissing":  map[string]any{"type": "boolean"},
				"providerDisabled": map[string]any{"type": "boolean"},
			},
			"required": []string{"name", "type", "listen", "port", "udp", "users", "userCount", "egressGroup", "status", "groupMissing", "providerMissing", "providerDisabled"},
		},
		"EventEntry": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":      map[string]any{"type": "string"},
				"level":   map[string]any{"type": "string"},
				"scope":   map[string]any{"type": "string"},
				"message": map[string]any{"type": "string"},
				"at":      map[string]any{"type": "string"},
			},
			"required": []string{"id", "level", "scope", "message", "at"},
		},
		"RuntimeState": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode":                map[string]any{"type": "string"},
				"mihomoBinary":        map[string]any{"type": "string"},
				"mihomoBinaryFound":   map[string]any{"type": "boolean"},
				"controller":          map[string]any{"type": "string"},
				"controllerReachable": map[string]any{"type": "boolean"},
				"configPath":          map[string]any{"type": "string"},
				"lastRenderAt":        map[string]any{"type": "string"},
				"lastApplyAt":         map[string]any{"type": "string"},
				"lastApplyStatus":     map[string]any{"type": "string"},
				"lastApplyMessage":    map[string]any{"type": "string"},
			},
			"required": []string{"mode", "mihomoBinaryFound", "controllerReachable"},
		},
		"ControllerState": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"reachable": map[string]any{"type": "boolean"},
				"mode":      map[string]any{"type": "string"},
				"version":   map[string]any{"type": "string"},
				"message":   map[string]any{"type": "string"},
				"checkedAt": map[string]any{"type": "string"},
			},
			"required": []string{"reachable", "mode", "message"},
		},
		"StatusApp": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":       map[string]any{"type": "string"},
				"mode":       map[string]any{"type": "string"},
				"startedAt":  map[string]any{"type": "string"},
				"renderOnly": map[string]any{"type": "boolean"},
			},
			"required": []string{"name", "mode", "startedAt", "renderOnly"},
		},
		"StatusCounts": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"providers": map[string]any{"type": "integer"},
				"groups":    map[string]any{"type": "integer"},
				"listeners": map[string]any{"type": "integer"},
				"nodes":     map[string]any{"type": "integer"},
			},
			"required": []string{"providers", "groups", "listeners", "nodes"},
		},
		"StatusResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"app":          schemaRef("StatusApp"),
				"runtime":      schemaRef("RuntimeState"),
				"controller":   schemaRef("ControllerState"),
				"counts":       schemaRef("StatusCounts"),
				"warnings":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"recentEvents": map[string]any{"type": "array", "items": schemaRef("EventEntry")},
			},
			"required": []string{"app", "runtime", "counts", "warnings", "recentEvents"},
		},
		"RenderedConfigResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":      map[string]any{"type": "string"},
				"updatedAt": map[string]any{"type": "string"},
				"content":   map[string]any{"type": "string"},
			},
			"required": []string{"path", "updatedAt", "content"},
		},
		"RuntimeCheck": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key":            map[string]any{"type": "string"},
				"title":          map[string]any{"type": "string"},
				"ok":             map[string]any{"type": "boolean"},
				"level":          map[string]any{"type": "string"},
				"summary":        map[string]any{"type": "string"},
				"recommendation": map[string]any{"type": "string"},
			},
			"required": []string{"key", "title", "ok", "level", "summary"},
		},
		"RuntimePreflightSummary": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ready":    map[string]any{"type": "boolean"},
				"total":    map[string]any{"type": "integer"},
				"passed":   map[string]any{"type": "integer"},
				"warnings": map[string]any{"type": "integer"},
				"errors":   map[string]any{"type": "integer"},
				"info":     map[string]any{"type": "integer"},
			},
			"required": []string{"ready", "total", "passed", "warnings", "errors", "info"},
		},
		"RuntimePreflightPaths": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"configPath":            map[string]any{"type": "string"},
				"runtimeDir":            map[string]any{"type": "string"},
				"providersDir":          map[string]any{"type": "string"},
				"mihomoConfigPath":      map[string]any{"type": "string"},
				"lastGoodConfigPath":    map[string]any{"type": "string"},
				"statePath":             map[string]any{"type": "string"},
				"proxyrelayServicePath": map[string]any{"type": "string"},
				"mihomoServicePath":     map[string]any{"type": "string"},
			},
			"required": []string{"configPath", "runtimeDir", "providersDir", "mihomoConfigPath", "lastGoodConfigPath", "statePath", "proxyrelayServicePath", "mihomoServicePath"},
		},
		"RuntimePreflightSnapshot": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"generatedAt":     map[string]any{"type": "string"},
				"mode":            map[string]any{"type": "string"},
				"status":          map[string]any{"type": "string"},
				"ready":           map[string]any{"type": "boolean"},
				"summary":         schemaRef("RuntimePreflightSummary"),
				"paths":           schemaRef("RuntimePreflightPaths"),
				"checks":          map[string]any{"type": "array", "items": schemaRef("RuntimeCheck")},
				"recommendations": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"generatedAt", "mode", "status", "ready", "summary", "paths", "checks", "recommendations"},
		},
		"SetupCheck": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key":     map[string]any{"type": "string"},
				"title":   map[string]any{"type": "string"},
				"ready":   map[string]any{"type": "boolean"},
				"summary": map[string]any{"type": "string"},
				"action":  map[string]any{"type": "string"},
			},
			"required": []string{"key", "title", "ready", "summary"},
		},
		"SetupState": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"required":             map[string]any{"type": "boolean"},
				"needsPasswordReset":   map[string]any{"type": "boolean"},
				"hasSubscriptions":     map[string]any{"type": "boolean"},
				"hasRealSubscriptions": map[string]any{"type": "boolean"},
				"hasEgressGroups":      map[string]any{"type": "boolean"},
				"hasListeners":         map[string]any{"type": "boolean"},
				"checks":               map[string]any{"type": "array", "items": schemaRef("SetupCheck")},
				"reasons":              map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"required", "needsPasswordReset", "hasSubscriptions", "hasRealSubscriptions", "hasEgressGroups", "hasListeners", "checks", "reasons"},
		},
		"ControllerStatusResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"baseUrl":          map[string]any{"type": "string"},
				"secretConfigured": map[string]any{"type": "boolean"},
				"renderOnly":       map[string]any{"type": "boolean"},
				"reachable":        map[string]any{"type": "boolean"},
				"mode":             map[string]any{"type": "string"},
				"version":          map[string]any{"type": "string"},
				"message":          map[string]any{"type": "string"},
				"checkedAt":        map[string]any{"type": "string"},
			},
			"required": []string{"baseUrl", "secretConfigured", "renderOnly", "reachable", "mode", "message"},
		},
		"ProbeControllerResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":         map[string]any{"type": "boolean"},
				"controller": schemaRef("ControllerStatusResponse"),
			},
			"required": []string{"ok", "controller"},
		},
		"ReloadResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":           map[string]any{"type": "boolean"},
				"message":      map[string]any{"type": "string"},
				"runtime":      schemaRef("RuntimeState"),
				"runtimeApply": schemaRef("RuntimeApplyResult"),
			},
			"required": []string{"ok", "message", "runtime", "runtimeApply"},
		},
		"RuntimeCommandResult": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":      map[string]any{"type": "boolean"},
				"status":  map[string]any{"type": "integer"},
				"error":   map[string]any{"type": "string"},
				"payload": map[string]any{"type": "object", "additionalProperties": true},
			},
		},
		"ProviderRefreshResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":       map[string]any{"type": "boolean"},
				"error":    map[string]any{"type": "string"},
				"provider": schemaRef("ProviderRecord"),
				"runtime":  schemaRef("RuntimeCommandResult"),
			},
			"required": []string{"ok"},
		},
		"SelectGroupRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"proxyName": map[string]any{"type": "string"},
			},
			"required": []string{"proxyName"},
		},
		"GroupMutationResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":      map[string]any{"type": "boolean"},
				"error":   map[string]any{"type": "string"},
				"group":   schemaRef("GroupView"),
				"runtime": schemaRef("RuntimeCommandResult"),
			},
			"required": []string{"ok"},
		},
		"ToggleSubscriptionRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"enabled": map[string]any{"type": "boolean"},
			},
			"required": []string{"enabled"},
		},
		"AddSubscriptionRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":                  map[string]any{"type": "string"},
				"url":                   map[string]any{"type": "string"},
				"type":                  map[string]any{"type": "string"},
				"interval":              map[string]any{"type": "integer"},
				"enabled":               map[string]any{"type": "boolean"},
				"health_check_url":      map[string]any{"type": "string"},
				"health_check_interval": map[string]any{"type": "integer"},
			},
			"required": []string{"name", "url"},
		},
		"AddEgressGroupRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":           map[string]any{"type": "string"},
				"provider":       map[string]any{"type": "string"},
				"mode":           map[string]any{"type": "string"},
				"filter":         map[string]any{"type": "string"},
				"exclude_filter": map[string]any{"type": "string"},
			},
			"required": []string{"name", "provider"},
		},
		"UpdateEgressGroupRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"provider":       map[string]any{"type": "string"},
				"mode":           map[string]any{"type": "string"},
				"filter":         map[string]any{"type": "string"},
				"exclude_filter": map[string]any{"type": "string"},
			},
		},
		"AddListenerRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":         map[string]any{"type": "string"},
				"type":         map[string]any{"type": "string"},
				"listen":       map[string]any{"type": "string"},
				"port":         map[string]any{"type": "integer"},
				"udp":          map[string]any{"type": "boolean"},
				"enabled":      map[string]any{"type": "boolean"},
				"users":        map[string]any{"type": "array", "items": schemaRef("ListenerUser")},
				"egress_group": map[string]any{"type": "string"},
			},
			"required": []string{"name", "port", "egress_group"},
		},
		"UpdateListenerRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type":         map[string]any{"type": "string"},
				"listen":       map[string]any{"type": "string"},
				"port":         map[string]any{"type": "integer"},
				"udp":          map[string]any{"type": "boolean"},
				"users":        map[string]any{"type": "array", "items": schemaRef("ListenerUser")},
				"egress_group": map[string]any{"type": "string"},
			},
		},
		"MihomoVersionSupport": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"version":         map[string]any{"type": "string"},
				"recommended":     map[string]any{"type": "boolean"},
				"supported":       map[string]any{"type": "boolean"},
				"compatibility":   map[string]any{"type": "string"},
				"releaseUrl":      map[string]any{"type": "string"},
				"assetUrl":        map[string]any{"type": "string"},
				"assetName":       map[string]any{"type": "string"},
				"currentPlatform": map[string]any{"type": "boolean"},
			},
			"required": []string{"version", "recommended", "supported", "compatibility", "releaseUrl", "currentPlatform"},
		},
		"MihomoVersionRecord": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"version":      map[string]any{"type": "string"},
				"status":       map[string]any{"type": "string"},
				"archivePath":  map[string]any{"type": "string"},
				"binaryPath":   map[string]any{"type": "string"},
				"downloadedAt": map[string]any{"type": "string"},
				"installedAt":  map[string]any{"type": "string"},
				"activatedAt":  map[string]any{"type": "string"},
				"lastError":    map[string]any{"type": "string"},
			},
			"required": []string{"version", "status"},
		},
		"MihomoVersionsResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"platform": map[string]any{
					"type":                 "object",
					"additionalProperties": map[string]any{"type": "string"},
				},
				"recommended":       map[string]any{"type": "string"},
				"activeVersion":     map[string]any{"type": "string"},
				"configuredBinary":  map[string]any{"type": "string"},
				"supportMatrix":     map[string]any{"type": "array", "items": schemaRef("MihomoVersionSupport")},
				"installedVersions": map[string]any{"type": "array", "items": schemaRef("MihomoVersionRecord")},
			},
			"required": []string{"platform", "recommended", "configuredBinary", "supportMatrix", "installedVersions"},
		},
		"MihomoVersionStateResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"records": map[string]any{"type": "array", "items": schemaRef("MihomoVersionRecord")},
			},
			"required": []string{"records"},
		},
		"MihomoVersionActionResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok":     map[string]any{"type": "boolean"},
				"error":  map[string]any{"type": "string"},
				"record": schemaRef("MihomoVersionRecord"),
			},
			"required": []string{"ok"},
		},
	}

	paths := map[string]any{
		"/api/openapi.json": map[string]any{
			"get": map[string]any{
				"summary":   "读取 OpenAPI 文档",
				"responses": map[string]any{"200": jsonResponse("OpenAPI 文档", map[string]any{"type": "object", "additionalProperties": true})},
			},
		},
		"/api/session": map[string]any{
			"get": map[string]any{
				"summary":   "查询会话",
				"responses": map[string]any{"200": jsonResponse("当前会话状态", schemaRef("SessionStatusResponse"))},
			},
			"post": map[string]any{
				"summary":     "登录",
				"requestBody": jsonBody(schemaRef("LoginRequest")),
				"responses": map[string]any{
					"200": jsonResponse("登录成功", schemaRef("LoginResponse")),
					"401": jsonResponse("账号或密码错误", schemaRef("ErrorResponse")),
				},
			},
			"delete": map[string]any{
				"summary":   "退出登录",
				"responses": map[string]any{"200": jsonResponse("退出成功", schemaRef("SimpleOkResponse"))},
			},
		},
		"/api/session/password": map[string]any{
			"put": map[string]any{
				"summary":     "修改密码",
				"requestBody": jsonBody(schemaRef("UpdatePasswordRequest")),
				"responses": map[string]any{
					"200": jsonResponse("密码已更新", schemaRef("SimpleOkResponse")),
					"400": jsonResponse("请求参数错误", schemaRef("ErrorResponse")),
					"401": jsonResponse("未登录或会话已过期", schemaRef("ErrorResponse")),
					"500": jsonResponse("服务端错误", schemaRef("ErrorResponse")),
				},
			},
		},
		"/api/status": map[string]any{
			"get": map[string]any{
				"summary":   "系统总览",
				"responses": map[string]any{"200": jsonResponse("系统状态概览", schemaRef("StatusResponse")), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/config": map[string]any{
			"get": map[string]any{
				"summary":   "读取配置",
				"responses": map[string]any{"200": jsonResponse("当前配置", schemaRef("ConfigEnvelope")), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
			"put": map[string]any{
				"summary":     "保存配置",
				"requestBody": jsonBody(schemaRef("SaveConfigRequest")),
				"responses": map[string]any{
					"200": jsonResponse("配置已保存", schemaRef("SaveConfigResponse")),
					"400": jsonResponse("请求参数错误", schemaRef("ErrorResponse")),
					"401": jsonResponse("未登录", schemaRef("ErrorResponse")),
					"500": jsonResponse("服务端错误", schemaRef("ErrorResponse")),
				},
			},
		},
		"/api/providers": map[string]any{
			"get": map[string]any{
				"summary":   "订阅列表",
				"responses": map[string]any{"200": jsonResponse("订阅与节点概览", map[string]any{"type": "array", "items": schemaRef("ProviderListItem")}), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/providers/{name}/refresh": map[string]any{
			"post": map[string]any{
				"summary":    "刷新订阅",
				"parameters": []map[string]any{pathParam("name", "订阅名称")},
				"responses": map[string]any{
					"200": jsonResponse("刷新结果", schemaRef("ProviderRefreshResponse")),
					"401": jsonResponse("未登录", schemaRef("ErrorResponse")),
					"404": jsonResponse("订阅不存在", schemaRef("ErrorResponse")),
				},
			},
		},
		"/api/groups": map[string]any{
			"get": map[string]any{
				"summary":   "出口组列表",
				"responses": map[string]any{"200": jsonResponse("出口组视图", map[string]any{"type": "array", "items": schemaRef("GroupView")}), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/groups/{groupName}/select": map[string]any{
			"post": map[string]any{
				"summary":     "切换出口组节点",
				"parameters":  []map[string]any{pathParam("groupName", "出口组名称")},
				"requestBody": jsonBody(schemaRef("SelectGroupRequest")),
				"responses": map[string]any{
					"200": jsonResponse("切换结果", schemaRef("GroupMutationResponse")),
					"400": jsonResponse("请求参数错误", schemaRef("ErrorResponse")),
					"401": jsonResponse("未登录", schemaRef("ErrorResponse")),
					"404": jsonResponse("出口组不存在", schemaRef("ErrorResponse")),
				},
			},
		},
		"/api/groups/{groupName}/healthcheck": map[string]any{
			"post": map[string]any{
				"summary":    "执行健康检查",
				"parameters": []map[string]any{pathParam("groupName", "出口组名称")},
				"responses": map[string]any{
					"200": jsonResponse("健康检查结果", schemaRef("GroupMutationResponse")),
					"401": jsonResponse("未登录", schemaRef("ErrorResponse")),
					"404": jsonResponse("出口组不存在", schemaRef("ErrorResponse")),
				},
			},
		},
		"/api/listeners": map[string]any{
			"get": map[string]any{
				"summary":   "监听器列表",
				"responses": map[string]any{"200": jsonResponse("监听器视图", map[string]any{"type": "array", "items": schemaRef("ListenerView")}), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
			"post": map[string]any{
				"summary":     "创建监听器",
				"requestBody": jsonBody(schemaRef("AddListenerRequest")),
				"responses":   map[string]any{"200": jsonResponse("保存结果", schemaRef("SaveConfigResponse"))},
			},
		},
		"/api/listeners/{name}/update": map[string]any{
			"put": map[string]any{
				"summary":     "更新监听器",
				"parameters":  []map[string]any{pathParam("name", "监听器名称")},
				"requestBody": jsonBody(schemaRef("UpdateListenerRequest")),
				"responses":   mutationErrorResponsesWith200("保存结果", "SaveConfigResponse"),
			},
		},
		"/api/listeners/{name}": map[string]any{
			"delete": map[string]any{
				"summary":    "删除监听器",
				"parameters": []map[string]any{pathParam("name", "监听器名称")},
				"responses":  mutationErrorResponsesWith200("保存结果", "SaveConfigResponse"),
			},
		},
		"/api/events": map[string]any{
			"get": map[string]any{
				"summary":   "事件日志",
				"responses": map[string]any{"200": jsonResponse("事件列表", map[string]any{"type": "array", "items": schemaRef("EventEntry")}), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/rendered-config": map[string]any{
			"get": map[string]any{
				"summary":   "查看当前生成的代理核心配置",
				"responses": map[string]any{"200": jsonResponse("当前生成结果", schemaRef("RenderedConfigResponse")), "401": jsonResponse("未登录", schemaRef("ErrorResponse")), "500": jsonResponse("生成结果读取失败", schemaRef("ErrorResponse"))},
			},
		},
		"/api/controller": map[string]any{
			"get": map[string]any{
				"summary":   "查看代理核心连接状态",
				"responses": map[string]any{"200": jsonResponse("代理核心连接结果", schemaRef("ControllerStatusResponse")), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/runtime-preflight": map[string]any{
			"get": map[string]any{
				"summary":   "查看使用前检查结果",
				"responses": map[string]any{"200": jsonResponse("使用前检查结果", schemaRef("RuntimePreflightSnapshot")), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/setup-state": map[string]any{
			"get": map[string]any{
				"summary":   "首次初始化状态",
				"responses": map[string]any{"200": jsonResponse("Setup 检查结果", schemaRef("SetupState")), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/controller/probe": map[string]any{
			"post": map[string]any{
				"summary":   "重新检测代理核心连接",
				"responses": map[string]any{"200": jsonResponse("检测结果", schemaRef("ProbeControllerResponse")), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/reload": map[string]any{
			"post": map[string]any{
				"summary":   "重新加载配置",
				"responses": map[string]any{"200": jsonResponse("重载结果", schemaRef("ReloadResponse")), "401": jsonResponse("未登录", schemaRef("ErrorResponse"))},
			},
		},
		"/api/subscriptions": map[string]any{
			"post": map[string]any{
				"summary":     "创建订阅",
				"requestBody": jsonBody(schemaRef("AddSubscriptionRequest")),
				"responses":   mutationErrorResponsesWith200("保存结果", "SaveConfigResponse"),
			},
		},
		"/api/subscriptions/{name}/toggle": map[string]any{
			"post": map[string]any{
				"summary":     "启停订阅",
				"parameters":  []map[string]any{pathParam("name", "订阅名称")},
				"requestBody": jsonBody(schemaRef("ToggleSubscriptionRequest")),
				"responses":   mutationErrorResponsesWith200("保存结果", "SaveConfigResponse"),
			},
		},
		"/api/subscriptions/{name}": map[string]any{
			"delete": map[string]any{
				"summary":    "删除订阅",
				"parameters": []map[string]any{pathParam("name", "订阅名称")},
				"responses":  mutationErrorResponsesWith200("保存结果", "SaveConfigResponse"),
			},
		},
		"/api/egress-groups": map[string]any{
			"post": map[string]any{
				"summary":     "创建出口组",
				"requestBody": jsonBody(schemaRef("AddEgressGroupRequest")),
				"responses":   mutationErrorResponsesWith200("保存结果", "SaveConfigResponse"),
			},
		},
		"/api/egress-groups/{name}/update": map[string]any{
			"put": map[string]any{
				"summary":     "更新出口组",
				"parameters":  []map[string]any{pathParam("name", "出口组名称")},
				"requestBody": jsonBody(schemaRef("UpdateEgressGroupRequest")),
				"responses":   mutationErrorResponsesWith200("保存结果", "SaveConfigResponse"),
			},
		},
		"/api/egress-groups/{name}": map[string]any{
			"delete": map[string]any{
				"summary":    "删除出口组",
				"parameters": []map[string]any{pathParam("name", "出口组名称")},
				"responses":  mutationErrorResponsesWith200("保存结果", "SaveConfigResponse"),
			},
		},
		"/api/mihomo/versions": map[string]any{
			"get": map[string]any{
				"summary":   "Mihomo 支持版本与安装状态",
				"responses": map[string]any{"200": jsonResponse("版本矩阵", schemaRef("MihomoVersionsResponse")), "401": jsonResponse("未登录", schemaRef("ErrorResponse")), "500": jsonResponse("服务端错误", schemaRef("ErrorResponse"))},
			},
		},
		"/api/mihomo/version-state": map[string]any{
			"get": map[string]any{
				"summary":   "Mihomo 版本任务状态",
				"responses": map[string]any{"200": jsonResponse("版本状态", schemaRef("MihomoVersionStateResponse")), "401": jsonResponse("未登录", schemaRef("ErrorResponse")), "500": jsonResponse("服务端错误", schemaRef("ErrorResponse"))},
			},
		},
		"/api/mihomo/versions/{version}/download": map[string]any{
			"post": map[string]any{
				"summary":    "下载 Mihomo 版本",
				"parameters": []map[string]any{pathParam("version", "目标版本")},
				"responses": map[string]any{
					"200": jsonResponse("下载结果", schemaRef("MihomoVersionActionResponse")),
					"400": jsonResponse("请求参数错误", schemaRef("ErrorResponse")),
					"401": jsonResponse("未登录", schemaRef("ErrorResponse")),
					"500": jsonResponse("服务端错误", schemaRef("ErrorResponse")),
				},
			},
		},
		"/api/mihomo/versions/{version}/install": map[string]any{
			"post": map[string]any{
				"summary":    "安装 Mihomo 版本",
				"parameters": []map[string]any{pathParam("version", "目标版本")},
				"responses": map[string]any{
					"200": jsonResponse("安装结果", schemaRef("MihomoVersionActionResponse")),
					"400": jsonResponse("请求参数错误", schemaRef("ErrorResponse")),
					"401": jsonResponse("未登录", schemaRef("ErrorResponse")),
					"500": jsonResponse("服务端错误", schemaRef("ErrorResponse")),
				},
			},
		},
		"/api/mihomo/versions/{version}/activate": map[string]any{
			"post": map[string]any{
				"summary":    "激活 Mihomo 版本",
				"parameters": []map[string]any{pathParam("version", "目标版本")},
				"responses": map[string]any{
					"200": jsonResponse("激活结果", schemaRef("MihomoVersionActionResponse")),
					"400": jsonResponse("请求参数错误", schemaRef("ErrorResponse")),
					"401": jsonResponse("未登录", schemaRef("ErrorResponse")),
					"500": jsonResponse("服务端错误", schemaRef("ErrorResponse")),
				},
			},
		},
	}

	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "AXIS API",
			"version":     "0.3.0",
			"description": "Go 原生控制面导出的 AXIS OpenAPI 文档",
		},
		"paths": paths,
		"components": map[string]any{
			"schemas": schemas,
		},
	}
}

func mutationErrorResponsesWith200(description, schemaName string) map[string]any {
	responses := mutationErrorResponses()
	responses["200"] = jsonResponse(description, schemaRef(schemaName))
	return responses
}
