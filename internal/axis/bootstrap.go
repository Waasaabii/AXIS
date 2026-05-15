package axis

func (s *Service) GetBootstrapStatus(valid bool, username string) BootstrapStatus {
	setup := s.GetSetupState()
	host := s.GetHostStatus()
	updater := s.GetUpdaterStatus()
	mainService := s.buildMainServiceStatus()

	auth := BootstrapAuthStatus{
		Authenticated:         valid,
		Username:              username,
		RequiresPasswordReset: s != nil && s.config != nil && s.config.Admin.RequiresPasswordReset,
	}

	nextStep, blockingReason := determineBootstrapNextStep(auth, setup, mainService)

	return BootstrapStatus{
		Host:           host,
		MainService:    mainService,
		Updater:        updater,
		Setup:          setup,
		Auth:           auth,
		NextStep:       nextStep,
		BlockingReason: blockingReason,
	}
}

func determineBootstrapNextStep(auth BootstrapAuthStatus, setup SetupState, mainService MainServiceStatus) (string, string) {
	if !mainService.Ready {
		return "launch", firstNonEmpty(mainService.BlockingReason, mainService.Message)
	}
	if setup.NeedsPasswordReset {
		return "setup", ""
	}
	if !auth.Authenticated {
		return "login", ""
	}
	return "dashboard", ""
}

func (s *Service) buildMainServiceStatus() MainServiceStatus {
	status := MainServiceStatus{
		Ready:   true,
		State:   "running",
		Message: "主服务已启动，可以继续进入控制台。",
	}
	if s == nil || s.config == nil || s.state == nil {
		status.Ready = false
		status.State = "error"
		status.Message = "主服务尚未完成初始化。"
		status.BlockingReason = "主服务配置未加载完成"
		return status
	}

	status.Mode = s.state.Runtime.Mode
	status.Controller = s.config.Runtime.ExternalController
	status.ConfigPath = s.configPath

	switch s.state.Runtime.State {
	case "render-only":
		status.State = "degraded"
		status.Message = s.state.Runtime.Message
	case "core-missing", "controller-unreachable", "apply-failed":
		status.State = "error"
		status.Message = s.state.Runtime.Message
	case "ready":
		status.State = "running"
		status.Message = s.state.Runtime.Message
	}

	return status
}
