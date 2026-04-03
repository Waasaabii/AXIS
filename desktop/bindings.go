package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Waasaabii/AXIS/internal/axis"
)

type desktopSession struct {
	mu            sync.RWMutex
	authenticated bool
	username      string
	fingerprint   string
}

func (s *desktopSession) status() (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authenticated, s.username
}

func (s *desktopSession) setAuthenticated(username string, fingerprint string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authenticated = true
	s.username = username
	s.fingerprint = fingerprint
}

func (s *desktopSession) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authenticated = false
	s.username = ""
	s.fingerprint = ""
}

func (s *desktopSession) invalidateIfFingerprintChanged(fingerprint string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authenticated {
		s.fingerprint = fingerprint
		return
	}
	if s.fingerprint == "" {
		s.fingerprint = fingerprint
		return
	}
	if fingerprint != "" && s.fingerprint != fingerprint {
		s.authenticated = false
		s.username = ""
		s.fingerprint = fingerprint
	}
}

type EngineBindings struct {
	engine  *axis.EngineFacade
	session *desktopSession
}

type HostBindings struct {
	host *axis.HostFacade
}

func NewEngineBindings(engine *axis.EngineFacade) *EngineBindings {
	return &EngineBindings{
		engine:  engine,
		session: &desktopSession{},
	}
}

func NewHostBindings(host *axis.HostFacade) *HostBindings {
	return &HostBindings{host: host}
}

func bindingError(result map[string]any, status int) error {
	if status < 400 {
		return nil
	}
	message := fmt.Sprintf("请求失败 (%d)", status)
	if value, ok := result["error"].(string); ok && value != "" {
		message = value
	} else if value, ok := result["message"].(string); ok && value != "" {
		message = value
	}
	return errors.New(message)
}

func (e *EngineBindings) GetSession() (map[string]any, error) {
	if err := e.syncSessionState(); err != nil {
		return nil, err
	}
	valid, username := e.session.status()
	return e.engine.GetSessionStatus(valid, username)
}

func (e *EngineBindings) GetBootstrapStatus() (axis.BootstrapStatus, error) {
	if err := e.syncSessionState(); err != nil {
		return axis.BootstrapStatus{}, err
	}
	valid, username := e.session.status()
	return e.engine.GetBootstrapStatus(valid, username)
}

func (e *EngineBindings) Login(payload map[string]any) (map[string]any, error) {
	username, _ := payload["username"].(string)
	password, _ := payload["password"].(string)
	result, status, err := e.engine.Login(username, password)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	fingerprint, err := e.engine.AuthFingerprint()
	if err != nil {
		return nil, err
	}
	e.session.setAuthenticated(username, fingerprint)
	return result, nil
}

func (e *EngineBindings) Logout() (map[string]any, error) {
	e.session.clear()
	return e.engine.Logout()
}

func (e *EngineBindings) UpdatePassword(payload map[string]any) (map[string]any, error) {
	password, _ := payload["password"].(string)
	result, status, err := e.engine.UpdatePassword(password)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	valid, username := e.session.status()
	if valid {
		fingerprint, fpErr := e.engine.AuthFingerprint()
		if fpErr != nil {
			return nil, fpErr
		}
		e.session.setAuthenticated(username, fingerprint)
	}
	return result, nil
}

func (e *EngineBindings) BootstrapAdmin(payload map[string]any) (map[string]any, error) {
	username, _ := payload["username"].(string)
	password, _ := payload["password"].(string)
	result, status, err := e.engine.BootstrapAdmin(username, password)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	e.session.clear()
	return result, nil
}

func (e *EngineBindings) GetStatus() (map[string]any, error) {
	return e.engine.GetStatus()
}

func (e *EngineBindings) GetConfig() (map[string]any, error) {
	return e.engine.GetConfig()
}

func (e *EngineBindings) SaveConfig(payload map[string]any) (map[string]any, error) {
	configPayload, ok := payload["config"].(map[string]any)
	if !ok {
		configPayload = payload
	}
	config, err := axis.DecodeConfigPayload(configPayload)
	if err != nil {
		return nil, err
	}
	result, status, saveErr := e.engine.SaveConfig(config)
	if saveErr != nil {
		return nil, saveErr
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetProviders() ([]map[string]any, error) {
	return e.engine.GetProviders()
}

func (e *EngineBindings) RefreshProvider(name string) (map[string]any, error) {
	return e.engine.RefreshProvider(name)
}

func (e *EngineBindings) GetGroups() ([]axis.GroupView, error) {
	return e.engine.GetGroups()
}

func (e *EngineBindings) GetTransitRoutes() ([]axis.TransitRouteView, error) {
	return e.engine.GetTransitRoutes()
}

func (e *EngineBindings) GetLandingProxies() ([]axis.LandingProxyView, error) {
	return e.engine.GetLandingProxies()
}

func (e *EngineBindings) SelectGroup(groupName string, payload map[string]any) (map[string]any, error) {
	proxyName, _ := payload["proxyName"].(string)
	result, status, err := e.engine.SelectGroup(groupName, proxyName)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) HealthcheckGroup(groupName string) (map[string]any, error) {
	result, status, err := e.engine.HealthcheckGroup(groupName)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) AddTransitRoute(payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.AddTransitRoute(payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateTransitRoute(name string, payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.UpdateTransitRoute(name, payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteTransitRoute(name string) (map[string]any, error) {
	result, status, err := e.engine.DeleteTransitRoute(name)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) HealthcheckTransitRoute(name string) (map[string]any, error) {
	result, status, err := e.engine.HealthcheckTransitRoute(name)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) AddLandingProxy(payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.AddLandingProxy(payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateLandingProxy(name string, payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.UpdateLandingProxy(name, payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteLandingProxy(name string) (map[string]any, error) {
	result, status, err := e.engine.DeleteLandingProxy(name)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetListeners() ([]axis.ListenerView, error) {
	return e.engine.GetListeners()
}

func (e *EngineBindings) AddListener(payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.AddListener(payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateListener(name string, payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.UpdateListener(name, payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteListener(name string) (map[string]any, error) {
	result, status, err := e.engine.DeleteListener(name)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetEvents() ([]axis.EventEntry, error) {
	return e.engine.GetEvents()
}

func (e *EngineBindings) GetRenderedConfig() (map[string]any, error) {
	return e.engine.GetRenderedConfig()
}

func (e *EngineBindings) GetController() (map[string]any, error) {
	return e.engine.GetControllerStatus()
}

func (e *EngineBindings) GetSetupState() (axis.SetupState, error) {
	return e.engine.GetSetupState()
}

func (e *EngineBindings) GetRuntimePreflight() (axis.RuntimePreflightSnapshot, error) {
	return e.engine.GetRuntimePreflight()
}

func (e *EngineBindings) ProbeController() (map[string]any, error) {
	return e.engine.ProbeController()
}

func (e *EngineBindings) ReloadRuntime() (map[string]any, error) {
	return e.engine.ReloadRuntime()
}

func (e *EngineBindings) AddSubscription(payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.AddSubscription(payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) ToggleSubscription(name string, payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.ToggleSubscription(name, payloadBool(payload, "enabled"))
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteSubscription(name string) (map[string]any, error) {
	result, status, err := e.engine.DeleteSubscription(name)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) AddEgressGroup(payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.AddEgressGroup(payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateEgressGroup(name string, payload map[string]any) (map[string]any, error) {
	result, status, err := e.engine.UpdateEgressGroup(name, payload)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteEgressGroup(name string) (map[string]any, error) {
	result, status, err := e.engine.DeleteEgressGroup(name)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetMihomoVersions() (axis.MihomoVersionsResponse, error) {
	return e.engine.GetMihomoVersions()
}

func (e *EngineBindings) DownloadMihomoVersion(version string) (map[string]any, error) {
	result, status, err := e.engine.DownloadMihomoVersion(version)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) InstallMihomoVersion(version string) (map[string]any, error) {
	result, status, err := e.engine.InstallMihomoVersion(version)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) ActivateMihomoVersion(version string) (map[string]any, error) {
	result, status, err := e.engine.ActivateMihomoVersion(version)
	if err != nil {
		return nil, err
	}
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *HostBindings) GetStatus() (axis.HostStatus, error) {
	return h.host.GetStatus()
}

func (h *HostBindings) GetUpdaterStatus() (axis.UpdaterStatus, error) {
	return h.host.GetUpdaterStatus()
}

func (h *HostBindings) OpenControlCenter() (map[string]any, error) {
	return h.host.OpenControlCenter("")
}

func (h *HostBindings) SetAutostart(enabled bool) (map[string]any, error) {
	return h.host.SetAutostart(enabled)
}

func (h *HostBindings) CheckForUpdates() (map[string]any, error) {
	return h.host.CheckForUpdates()
}

func payloadBool(payload map[string]any, key string) bool {
	value, ok := payload[key].(bool)
	return ok && value
}

func (e *EngineBindings) syncSessionState() error {
	if e == nil || e.engine == nil || e.session == nil {
		return nil
	}
	reloaded, err := e.engine.RefreshExternalState()
	if err != nil {
		return err
	}
	if reloaded {
		e.session.clear()
	}
	fingerprint, err := e.engine.AuthFingerprint()
	if err != nil {
		return err
	}
	e.session.invalidateIfFingerprintChanged(fingerprint)
	return nil
}
