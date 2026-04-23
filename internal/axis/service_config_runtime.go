package axis

import (
	"fmt"
	"os"
	"strings"
)

func (s *Service) cloneConfigForResponse() Config {
	clone := s.cloneConfig()
	clone.Admin.Password = ""
	clone.Admin.PasswordHash = ""
	clone.Admin.SessionSecret = ""
	return clone
}

func (s *Service) mergeAdminSecrets(next *Config) *Config {
	out := mustJSONClone(*next)
	out.Admin.Password = s.config.Admin.Password
	out.Admin.PasswordHash = s.config.Admin.PasswordHash
	out.Admin.SessionSecret = s.config.Admin.SessionSecret
	return &out
}

func (s *Service) GetConfig() map[string]any {
	return map[string]any{
		"path":   s.configPath,
		"config": s.cloneConfigForResponse(),
	}
}

func (s *Service) SaveConfig(next *Config) (map[string]any, int) {
	if next == nil {
		return map[string]any{"ok": false, "error": "config 不能为空"}, 400
	}
	merged := s.mergeAdminSecrets(next)
	if _, err := WriteConfig(s.configPath, merged); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, 400
	}
	runtimeApply, err := s.reloadAndApplyConfig("config", "配置已保存并重新加载", "配置保存")
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	return map[string]any{
		"ok":           true,
		"path":         s.configPath,
		"config":       s.cloneConfigForResponse(),
		"runtimeApply": runtimeApply,
	}, 200
}

func (s *Service) SaveConfigFromAPI(next *Config) (map[string]any, int) {
	if next == nil {
		return map[string]any{"ok": false, "error": "config 不能为空"}, 400
	}

	// 强制“唯一入口”：管理员密码和会话密钥只能走专用接口，不允许通过高级配置绕过。
	if strings.TrimSpace(next.Admin.Password) != "" ||
		strings.TrimSpace(next.Admin.PasswordHash) != "" ||
		strings.TrimSpace(next.Admin.SessionSecret) != "" {
		return map[string]any{"ok": false, "error": "不允许通过高级配置修改管理员密码或会话密钥，请使用“管理员密码”页面完成操作。"}, 400
	}

	return s.SaveConfig(next)
}

func (s *Service) GetRenderedConfig() (map[string]any, error) {
	content, err := os.ReadFile(s.layout.MihomoConfigPath)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"path":      s.layout.MihomoConfigPath,
		"updatedAt": s.state.Runtime.LastRenderAt,
		"content":   string(content),
	}, nil
}

func (s *Service) GetRuntimePreflight() RuntimePreflightSnapshot {
	s.detectController(false)
	return BuildRuntimePreflightSnapshot(s.configPath, s.config, s.layout, s.state, s.state.Controller)
}

func (s *Service) GetControllerStatus() map[string]any {
	s.detectController(false)
	if s.state.Controller == nil {
		return map[string]any{
			"baseUrl":          s.config.Runtime.ExternalController,
			"secretConfigured": s.config.Runtime.ExternalSecret != "",
			"renderOnly":       s.config.Runtime.RenderOnly,
			"reachable":        false,
			"mode":             ternaryString(s.config.Runtime.RenderOnly, "render-only", "managed"),
			"message":          "尚未检测",
		}
	}
	return map[string]any{
		"baseUrl":          s.config.Runtime.ExternalController,
		"secretConfigured": s.config.Runtime.ExternalSecret != "",
		"renderOnly":       s.config.Runtime.RenderOnly,
		"reachable":        s.state.Controller.Reachable,
		"mode":             s.state.Controller.Mode,
		"version":          s.state.Controller.Version,
		"message":          s.state.Controller.Message,
		"checkedAt":        s.state.Controller.CheckedAt,
	}
}

func (s *Service) ProbeController() map[string]any {
	s.detectController(true)
	_ = s.persistState()
	return map[string]any{
		"ok":         s.state.Controller != nil && s.state.Controller.Reachable,
		"controller": s.GetControllerStatus(),
	}
}

func (s *Service) ReloadConfig() map[string]any {
	runtimeApply, err := s.reloadAndApplyConfig("reload", "配置已重新加载并渲染", "配置重载")
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{
		"ok":           true,
		"message":      summarizeRuntimeApplyResult(runtimeApply, "配置已重新加载并应用到代理核心"),
		"runtime":      s.state.Runtime,
		"runtimeApply": runtimeApply,
	}
}

func (s *Service) reloadAndApplyConfig(scope, eventMessage, applyReason string) (map[string]any, error) {
	if err := s.LoadAll(); err != nil {
		return nil, err
	}
	s.pushEvent("info", scope, eventMessage)
	runtimeApply := s.applyRuntimeConfig(applyReason)
	_ = s.persistState()
	return runtimeApply, nil
}

func summarizeRuntimeApplyResult(runtimeApply map[string]any, successMessage string) string {
	if deferred, ok := runtimeApply["deferred"].(bool); ok && deferred {
		return fmt.Sprintf("配置已重新加载：%v", runtimeApply["message"])
	}
	if ok, ok2 := runtimeApply["ok"].(bool); ok2 && !ok {
		return fmt.Sprintf("配置已重新加载，但应用到代理核心失败：%v", runtimeApply["message"])
	}
	return successMessage
}

func (s *Service) readControllerResultMessage(result *controllerResult, fallback string) string {
	if result == nil {
		return fallback
	}
	switch payload := result.Payload.(type) {
	case string:
		if strings.TrimSpace(payload) != "" {
			return strings.TrimSpace(payload)
		}
	case map[string]any:
		if value, ok := payload["message"].(string); ok && value != "" {
			return value
		}
		if value, ok := payload["error"].(string); ok && value != "" {
			return value
		}
	}
	return fallback
}

func (s *Service) applyRuntimeConfig(reason string) map[string]any {
	appliedAt := nowISO()
	if s.config.Runtime.RenderOnly {
		message := "当前只保存配置，暂不会自动应用到代理核心。"
		s.state.Runtime.LastApplyAt = appliedAt
		s.state.Runtime.LastApplyStatus = "deferred"
		s.state.Runtime.LastApplyMessage = message
		_ = s.persistState()
		return map[string]any{"ok": false, "deferred": true, "message": message}
	}

	s.detectController(false)
	result, err := s.controller.ReloadConfig(s.layout.MihomoConfigPath)
	if err != nil {
		message := err.Error()
		s.state.Runtime.LastApplyAt = appliedAt
		s.state.Runtime.LastApplyStatus = "failed"
		s.state.Runtime.LastApplyMessage = message
		s.pushEvent("error", "runtime-apply", fmt.Sprintf("%s: %s", reason, message))
		_ = s.persistState()
		return map[string]any{"ok": false, "message": message}
	}

	message := s.readControllerResultMessage(result, fmt.Sprintf("代理核心返回状态码 %d", result.Status))
	if result.OK {
		message = "最新配置已发送到代理核心。"
		s.syncTransitMirrorSelections("", "")
	}
	s.state.Runtime.LastApplyAt = appliedAt
	if result.OK {
		s.state.Runtime.LastApplyStatus = "success"
		s.pushEvent("info", "runtime-apply", fmt.Sprintf("%s: %s", reason, message))
	} else {
		s.state.Runtime.LastApplyStatus = "failed"
		s.pushEvent("warn", "runtime-apply", fmt.Sprintf("%s: %s", reason, message))
	}
	s.state.Runtime.LastApplyMessage = message
	_ = s.persistState()
	return map[string]any{"ok": result.OK, "status": result.Status, "message": message}
}

func (s *Service) RefreshIfConfigChanged() (bool, error) {
	if s == nil || strings.TrimSpace(s.configPath) == "" {
		return false, nil
	}
	stat, err := os.Stat(s.configPath)
	if err != nil {
		return false, err
	}
	if !stat.ModTime().Equal(s.configMTime) {
		return true, s.LoadAll()
	}
	return false, nil
}

func (s *Service) captureConfigModTime() error {
	stat, err := os.Stat(s.configPath)
	if err != nil {
		return err
	}
	s.configMTime = stat.ModTime()
	return nil
}
