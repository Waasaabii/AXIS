package axis

import (
	"runtime"
	"strconv"
)

func normalizeState(state *AppState) *AppState {
	if state == nil {
		return createEmptyState()
	}
	if state.Providers == nil {
		state.Providers = map[string]ProviderRecord{}
	}
	if state.GroupSelections == nil {
		state.GroupSelections = map[string]string{}
	}
	if state.Groups == nil {
		state.Groups = map[string]GroupState{}
	}
	if state.TransitRoutes == nil {
		state.TransitRoutes = map[string]TransitRouteState{}
	}
	return state
}

func (s *Service) SetHostIntegration(host HostIntegration) {
	if host == nil {
		host = NewNoopHostIntegration()
	}
	s.host = host
	if s.state != nil {
		s.state.Host.DesktopMode = host.Mode() == "desktop"
		s.state.Host.AutostartEnabled = host.AutostartEnabled()
	}
}

func (s *Service) GetHostStatus() HostStatus {
	host := s.host
	if host == nil {
		host = NewNoopHostIntegration()
	}

	listenAddress := ""
	if s.config != nil {
		listenAddress = s.config.Server.Host
		if s.config.Server.Port > 0 {
			listenAddress += ":" + strconv.Itoa(s.config.Server.Port)
		}
	}

	status := HostStatus{
		Mode:             host.Mode(),
		Platform:         runtime.GOOS,
		DesktopMode:      host.Mode() == "desktop",
		AutostartEnabled: host.AutostartEnabled(),
		AutostartManaged: host.AutostartManaged(),
		ConfigPath:       s.configPath,
		ListenAddress:    listenAddress,
		Logs:             host.RecentLogs(20),
	}
	if s.layout != nil {
		status.RuntimeDir = s.layout.RuntimeDir
	}
	if s.state != nil {
		s.state.Host.DesktopMode = status.DesktopMode
		s.state.Host.AutostartEnabled = status.AutostartEnabled
	}
	return status
}

func (s *Service) GetUpdaterStatus() UpdaterStatus {
	host := s.host
	if host == nil {
		host = NewNoopHostIntegration()
	}
	status := host.UpdaterStatus()
	if status.Mode == "" {
		status.Mode = host.Mode()
	}
	if status.State == "" {
		status.State = "idle"
	}
	return status
}

func (s *Service) SetHostAutostart(enabled bool) (map[string]any, error) {
	host := s.host
	if host == nil {
		host = NewNoopHostIntegration()
	}
	if err := host.SetAutostart(enabled); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, err
	}
	if s.state != nil {
		s.state.Host.AutostartEnabled = enabled
		_ = s.persistState()
	}
	return map[string]any{"ok": true}, nil
}

func (s *Service) CheckForUpdates() (map[string]any, error) {
	host := s.host
	if host == nil {
		host = NewNoopHostIntegration()
	}
	if err := host.CheckForUpdates(); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, err
	}
	return map[string]any{"ok": true}, nil
}

func (s *Service) OpenHostControlCenter(url string) (map[string]any, error) {
	host := s.host
	if host == nil {
		host = NewNoopHostIntegration()
	}
	if err := host.OpenControlCenter(url); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, err
	}
	if s.state != nil {
		s.state.Host.LastOpenedAt = nowISO()
		_ = s.persistState()
	}
	return map[string]any{"ok": true}, nil
}

func (s *Service) RuntimeDir() string {
	if s == nil || s.layout == nil {
		return ""
	}
	return s.layout.RuntimeDir
}
