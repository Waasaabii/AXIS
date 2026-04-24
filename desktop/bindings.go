package main

import (
	"errors"
	"fmt"
	"strings"
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
	service *axis.Service
	session *desktopSession
}

type HostBindings struct {
	service *axis.Service
}

func NewEngineBindings(service *axis.Service) *EngineBindings {
	return &EngineBindings{
		service: service,
		session: &desktopSession{},
	}
}

func NewHostBindings(service *axis.Service) *HostBindings {
	return &HostBindings{service: service}
}

func (e *EngineBindings) serviceOrError() (*axis.Service, error) {
	if e == nil || e.service == nil {
		return nil, errors.New("service 未初始化")
	}
	return e.service, nil
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
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.SessionStatus(valid, username), nil
}

func (e *EngineBindings) GetBootstrapStatus() (axis.BootstrapStatus, error) {
	if err := e.syncSessionState(); err != nil {
		return axis.BootstrapStatus{}, err
	}
	valid, username := e.session.status()
	service, err := e.serviceOrError()
	if err != nil {
		return axis.BootstrapStatus{}, err
	}
	return service.GetBootstrapStatus(valid, username), nil
}

func (e *EngineBindings) Login(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	username, _ := payload["username"].(string)
	password, _ := payload["password"].(string)
	result, status := service.Login(username, password)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	fingerprint := service.AuthFingerprint()
	e.session.setAuthenticated(username, fingerprint)
	return result, nil
}

func (e *EngineBindings) Logout() (map[string]any, error) {
	e.session.clear()
	return map[string]any{"ok": true}, nil
}

func (e *EngineBindings) UpdatePassword(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	password, _ := payload["password"].(string)
	result, status := service.UpdatePassword(password)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	valid, username := e.session.status()
	if valid {
		fingerprint := service.AuthFingerprint()
		e.session.setAuthenticated(username, fingerprint)
	}
	return result, nil
}

func (e *EngineBindings) BootstrapAdmin(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	username, _ := payload["username"].(string)
	password, _ := payload["password"].(string)
	result, status := service.BootstrapAdmin(username, password)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	e.session.clear()
	return result, nil
}

func (e *EngineBindings) GetStatus() (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetStatus(), nil
}

func (e *EngineBindings) GetConfig() (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetConfig(), nil
}

func (e *EngineBindings) SaveConfig(payload map[string]any) (map[string]any, error) {
	config, err := axis.DecodeConfigPayload(payload)
	if err != nil {
		return nil, err
	}
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.SaveConfigFromAPI(config)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetNodeSources() ([]axis.NodeSource, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetNodeSources(), nil
}

func (e *EngineBindings) AddNodeSource(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.AddNodeSource(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateNodeSource(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdateNodeSource(name, payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteNodeSource(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RemoveNodeSource(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) CheckLocalNode(name string) (axis.LocalNodeCheckResponse, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return axis.LocalNodeCheckResponse{}, err
	}
	result, status := service.CheckLocalNode(name)
	if err := bindingError(map[string]any{"ok": result.OK, "error": result.Message}, status); err != nil {
		return axis.LocalNodeCheckResponse{}, err
	}
	return result, nil
}

func (e *EngineBindings) GetLocalNodeConnection(name string) (axis.LocalNodeConnectionResponse, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return axis.LocalNodeConnectionResponse{}, err
	}
	result, status := service.GetLocalNodeConnection(name)
	if err := bindingError(map[string]any{"ok": result.OK, "error": strings.Join(result.Warnings, "；")}, status); err != nil {
		return axis.LocalNodeConnectionResponse{}, err
	}
	return result, nil
}

func (e *EngineBindings) GetLocalNodeBaotaConfig(name string) (axis.BaotaConfigResponse, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return axis.BaotaConfigResponse{}, err
	}
	result, status := service.GetLocalNodeBaotaConfig(name)
	if err := bindingError(map[string]any{"ok": result.OK, "error": strings.Join(result.Warnings, "；")}, status); err != nil {
		return axis.BaotaConfigResponse{}, err
	}
	return result, nil
}

func (e *EngineBindings) GetRoutes() ([]axis.RouteConfig, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetRoutes(), nil
}

func (e *EngineBindings) AddRoute(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.AddRoute(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateRoute(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdateRoute(name, payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteRoute(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RemoveRoute(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetUsage() (axis.UsageView, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return axis.UsageView{}, err
	}
	return service.GetUsage(), nil
}

func (e *EngineBindings) UpdateUsage(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdateUsage(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetPublications() ([]axis.PublicationConfig, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetPublications(), nil
}

func (e *EngineBindings) AddPublication(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.AddPublication(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdatePublication(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdatePublication(name, payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeletePublication(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RemovePublication(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetProviders() ([]map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetProviders(), nil
}

func (e *EngineBindings) RefreshProvider(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.RefreshProvider(name), nil
}

func (e *EngineBindings) GetGroups() ([]axis.GroupView, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetGroups(), nil
}

func (e *EngineBindings) GetTransitRoutes() ([]axis.TransitRouteView, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetTransitRoutes(), nil
}

func (e *EngineBindings) GetLandingProxies() ([]axis.LandingProxyView, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetLandingProxies(), nil
}

func (e *EngineBindings) SelectGroup(groupName string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	proxyName, _ := payload["proxyName"].(string)
	result, status := service.SelectGroup(groupName, proxyName)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) HealthcheckGroup(groupName string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RunHealthcheck(groupName)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) AddTransitRoute(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.AddTransitRoute(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateTransitRoute(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdateTransitRoute(name, payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteTransitRoute(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RemoveTransitRoute(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) HealthcheckTransitRoute(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RunTransitRouteHealthcheck(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) AddLandingProxy(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.AddLandingProxy(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateLandingProxy(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdateLandingProxy(name, payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteLandingProxy(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RemoveLandingProxy(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetListeners() ([]axis.ListenerView, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetListeners(), nil
}

func (e *EngineBindings) AddListener(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.AddListener(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateListener(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdateListener(name, payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteListener(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RemoveListener(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetEvents() ([]axis.EventEntry, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetEvents(), nil
}

func (e *EngineBindings) GetRenderedConfig() (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetRenderedConfig()
}

func (e *EngineBindings) GetController() (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetControllerStatus(), nil
}

func (e *EngineBindings) GetSetupState() (axis.SetupState, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return axis.SetupState{}, err
	}
	return service.GetSetupState(), nil
}

func (e *EngineBindings) GetRuntimePreflight() (axis.RuntimePreflightSnapshot, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return axis.RuntimePreflightSnapshot{}, err
	}
	return service.GetRuntimePreflight(), nil
}

func (e *EngineBindings) ProbeController() (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.ProbeController(), nil
}

func (e *EngineBindings) ReloadRuntime() (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.ReloadConfig(), nil
}

func (e *EngineBindings) AddSubscription(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.AddSubscription(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateSubscription(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdateSubscription(name, payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) ToggleSubscription(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.ToggleSubscription(name, payloadBool(payload, "enabled"))
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteSubscription(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RemoveSubscription(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) AddEgressGroup(payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.AddEgressGroup(payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) UpdateEgressGroup(name string, payload map[string]any) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.UpdateEgressGroup(name, payload)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) DeleteEgressGroup(name string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.RemoveEgressGroup(name)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) GetMihomoVersions() (axis.MihomoVersionsResponse, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return axis.MihomoVersionsResponse{}, err
	}
	return service.MihomoVersions()
}

func (e *EngineBindings) DownloadMihomoVersion(version string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.DownloadMihomoVersion(version)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) InstallMihomoVersion(version string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.InstallMihomoVersion(version)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *EngineBindings) ActivateMihomoVersion(version string) (map[string]any, error) {
	service, err := e.serviceOrError()
	if err != nil {
		return nil, err
	}
	result, status := service.ActivateMihomoVersion(version)
	if err := bindingError(result, status); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *HostBindings) GetStatus() (axis.HostStatus, error) {
	if h == nil || h.service == nil {
		return axis.HostStatus{}, errors.New("service 未初始化")
	}
	return h.service.GetHostStatus(), nil
}

func (h *HostBindings) GetUpdaterStatus() (axis.UpdaterStatus, error) {
	if h == nil || h.service == nil {
		return axis.UpdaterStatus{}, errors.New("service 未初始化")
	}
	return h.service.GetUpdaterStatus(), nil
}

func (h *HostBindings) OpenControlCenter() (map[string]any, error) {
	if h == nil || h.service == nil {
		return nil, errors.New("service 未初始化")
	}
	return h.service.OpenHostControlCenter("")
}

func (h *HostBindings) SetAutostart(enabled bool) (map[string]any, error) {
	if h == nil || h.service == nil {
		return nil, errors.New("service 未初始化")
	}
	return h.service.SetHostAutostart(enabled)
}

func (h *HostBindings) CheckForUpdates() (map[string]any, error) {
	if h == nil || h.service == nil {
		return nil, errors.New("service 未初始化")
	}
	return h.service.CheckForUpdates()
}

func payloadBool(payload map[string]any, key string) bool {
	value, ok := payload[key].(bool)
	return ok && value
}

func (e *EngineBindings) syncSessionState() error {
	if e == nil || e.session == nil {
		return nil
	}
	service, err := e.serviceOrError()
	if err != nil {
		return err
	}
	reloaded, err := service.RefreshIfConfigChanged()
	if err != nil {
		return err
	}
	if reloaded {
		e.session.clear()
	}
	fingerprint := service.AuthFingerprint()
	e.session.invalidateIfFingerprintChanged(fingerprint)
	return nil
}
