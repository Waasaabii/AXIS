package axis

import "fmt"

type EngineFacade struct {
	service *Service
}

type HostFacade struct {
	service *Service
}

func NewEngineFacade(service *Service) *EngineFacade {
	return &EngineFacade{service: service}
}

func NewHostFacade(service *Service) *HostFacade {
	return &HostFacade{service: service}
}

func (f *EngineFacade) serviceOrError() (*Service, error) {
	if f == nil || f.service == nil {
		return nil, fmt.Errorf("service 未初始化")
	}
	return f.service, nil
}

func (f *HostFacade) serviceOrError() (*Service, error) {
	if f == nil || f.service == nil {
		return nil, fmt.Errorf("service 未初始化")
	}
	return f.service, nil
}

func (f *EngineFacade) GetSessionStatus(valid bool, username string) (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.SessionStatus(valid, username), nil
}

func (f *EngineFacade) RefreshExternalState() (bool, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return false, err
	}
	return service.RefreshIfConfigChanged()
}

func (f *EngineFacade) AuthFingerprint() (string, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return "", err
	}
	return service.AuthFingerprint(), nil
}

func (f *EngineFacade) GetBootstrapStatus(valid bool, username string) (BootstrapStatus, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return BootstrapStatus{}, err
	}
	return service.GetBootstrapStatus(valid, username), nil
}

func (f *EngineFacade) Login(username, password string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.Login(username, password)
	return result, status, nil
}

func (f *EngineFacade) Logout() (map[string]any, error) {
	if _, err := f.serviceOrError(); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

func (f *EngineFacade) UpdatePassword(password string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.UpdatePassword(password)
	return result, status, nil
}

func (f *EngineFacade) BootstrapAdmin(username, password string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.BootstrapAdmin(username, password)
	return result, status, nil
}

func (f *EngineFacade) GetStatus() (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetStatus(), nil
}

func (f *EngineFacade) GetConfig() (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetConfig(), nil
}

func (f *EngineFacade) SaveConfig(next *Config) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.SaveConfig(next)
	return result, status, nil
}

func (f *EngineFacade) GetProviders() ([]map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetProviders(), nil
}

func (f *EngineFacade) RefreshProvider(name string) (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.RefreshProvider(name), nil
}

func (f *EngineFacade) GetGroups() ([]GroupView, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetGroups(), nil
}

func (f *EngineFacade) GetTransitRoutes() ([]TransitRouteView, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetTransitRoutes(), nil
}

func (f *EngineFacade) GetLandingProxies() ([]LandingProxyView, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetLandingProxies(), nil
}

func (f *EngineFacade) SelectGroup(groupName, proxyName string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.SelectGroup(groupName, proxyName)
	return result, status, nil
}

func (f *EngineFacade) HealthcheckGroup(groupName string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.RunHealthcheck(groupName)
	return result, status, nil
}

func (f *EngineFacade) AddTransitRoute(payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.AddTransitRoute(payload)
	return result, status, nil
}

func (f *EngineFacade) UpdateTransitRoute(name string, payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.UpdateTransitRoute(name, payload)
	return result, status, nil
}

func (f *EngineFacade) DeleteTransitRoute(name string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.RemoveTransitRoute(name)
	return result, status, nil
}

func (f *EngineFacade) HealthcheckTransitRoute(name string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.RunTransitRouteHealthcheck(name)
	return result, status, nil
}

func (f *EngineFacade) AddLandingProxy(payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.AddLandingProxy(payload)
	return result, status, nil
}

func (f *EngineFacade) UpdateLandingProxy(name string, payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.UpdateLandingProxy(name, payload)
	return result, status, nil
}

func (f *EngineFacade) DeleteLandingProxy(name string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.RemoveLandingProxy(name)
	return result, status, nil
}

func (f *EngineFacade) GetListeners() ([]ListenerView, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetListeners(), nil
}

func (f *EngineFacade) AddListener(payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.AddListener(payload)
	return result, status, nil
}

func (f *EngineFacade) UpdateListener(name string, payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.UpdateListener(name, payload)
	return result, status, nil
}

func (f *EngineFacade) DeleteListener(name string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.RemoveListener(name)
	return result, status, nil
}

func (f *EngineFacade) GetEvents() ([]EventEntry, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetEvents(), nil
}

func (f *EngineFacade) GetRenderedConfig() (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetRenderedConfig()
}

func (f *EngineFacade) GetControllerStatus() (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.GetControllerStatus(), nil
}

func (f *EngineFacade) GetSetupState() (SetupState, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return SetupState{}, err
	}
	return service.GetSetupState(), nil
}

func (f *EngineFacade) GetRuntimePreflight() (RuntimePreflightSnapshot, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return RuntimePreflightSnapshot{}, err
	}
	return service.GetRuntimePreflight(), nil
}

func (f *EngineFacade) ProbeController() (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.ProbeController(), nil
}

func (f *EngineFacade) ReloadRuntime() (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.ReloadConfig(), nil
}

func (f *EngineFacade) AddSubscription(payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.AddSubscription(payload)
	return result, status, nil
}

func (f *EngineFacade) ToggleSubscription(name string, enabled bool) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.ToggleSubscription(name, enabled)
	return result, status, nil
}

func (f *EngineFacade) DeleteSubscription(name string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.RemoveSubscription(name)
	return result, status, nil
}

func (f *EngineFacade) AddEgressGroup(payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.AddEgressGroup(payload)
	return result, status, nil
}

func (f *EngineFacade) UpdateEgressGroup(name string, payload map[string]any) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.UpdateEgressGroup(name, payload)
	return result, status, nil
}

func (f *EngineFacade) DeleteEgressGroup(name string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.RemoveEgressGroup(name)
	return result, status, nil
}

func (f *EngineFacade) GetMihomoVersions() (MihomoVersionsResponse, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return MihomoVersionsResponse{}, err
	}
	return service.MihomoVersions()
}

func (f *EngineFacade) DownloadMihomoVersion(version string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.DownloadMihomoVersion(version)
	return result, status, nil
}

func (f *EngineFacade) InstallMihomoVersion(version string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.InstallMihomoVersion(version)
	return result, status, nil
}

func (f *EngineFacade) ActivateMihomoVersion(version string) (map[string]any, int, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, 500, err
	}
	result, status := service.ActivateMihomoVersion(version)
	return result, status, nil
}

func (f *HostFacade) GetStatus() (HostStatus, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return HostStatus{}, err
	}
	return service.GetHostStatus(), nil
}

func (f *HostFacade) GetUpdaterStatus() (UpdaterStatus, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return UpdaterStatus{}, err
	}
	return service.GetUpdaterStatus(), nil
}

func (f *HostFacade) OpenControlCenter(url string) (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.OpenHostControlCenter(url)
}

func (f *HostFacade) SetAutostart(enabled bool) (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.SetHostAutostart(enabled)
}

func (f *HostFacade) CheckForUpdates() (map[string]any, error) {
	service, err := f.serviceOrError()
	if err != nil {
		return nil, err
	}
	return service.CheckForUpdates()
}
