package axis

import "fmt"

type HostIntegration interface {
	Mode() string
	AutostartEnabled() bool
	AutostartManaged() bool
	SetAutostart(enabled bool) error
	OpenControlCenter(url string) error
	RecentLogs(limit int) []string
	UpdaterStatus() UpdaterStatus
	CheckForUpdates() error
	Shutdown() error
}

type NoopHostIntegration struct{}

func NewNoopHostIntegration() HostIntegration {
	return NoopHostIntegration{}
}

func (NoopHostIntegration) Mode() string {
	return "web"
}

func (NoopHostIntegration) AutostartEnabled() bool {
	return false
}

func (NoopHostIntegration) AutostartManaged() bool {
	return false
}

func (NoopHostIntegration) SetAutostart(bool) error {
	return fmt.Errorf("当前运行模式不支持开机自启")
}

func (NoopHostIntegration) OpenControlCenter(string) error {
	return fmt.Errorf("当前运行模式不支持打开本地控制台")
}

func (NoopHostIntegration) RecentLogs(limit int) []string {
	if limit <= 0 {
		return nil
	}
	return nil
}

func (NoopHostIntegration) UpdaterStatus() UpdaterStatus {
	return UpdaterStatus{
		Mode:    "unsupported",
		Running: false,
		State:   "idle",
		Message: "当前运行模式未启用桌面更新服务",
	}
}

func (NoopHostIntegration) CheckForUpdates() error {
	return fmt.Errorf("当前运行模式未启用桌面更新服务")
}

func (NoopHostIntegration) Shutdown() error {
	return nil
}
