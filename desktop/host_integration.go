package main

import (
	"context"
	"os"
	"sync"

	"github.com/Waasaabii/AXIS/internal/axis"
	"github.com/emersion/go-autostart"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type DesktopHostIntegration struct {
	mu        sync.Mutex
	ctx       context.Context
	autostart *autostart.App
	logs      []string
	updater   *UpdaterManager
}

func NewDesktopHostIntegration() *DesktopHostIntegration {
	executable, _ := os.Executable()
	return &DesktopHostIntegration{
		autostart: &autostart.App{
			Name:        "com.waasaabii.axis",
			DisplayName: "AXIS",
			Exec:        []string{executable},
		},
		logs: []string{},
	}
}

func (d *DesktopHostIntegration) SetUpdater(updater *UpdaterManager) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.updater = updater
}

func (d *DesktopHostIntegration) SetContext(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ctx = ctx
}

func (d *DesktopHostIntegration) appendLog(message string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logs = append([]string{message}, d.logs...)
	if len(d.logs) > 100 {
		d.logs = d.logs[:100]
	}
}

func (d *DesktopHostIntegration) Mode() string {
	return "desktop"
}

func (d *DesktopHostIntegration) AutostartEnabled() bool {
	if d.autostart == nil {
		return false
	}
	return d.autostart.IsEnabled()
}

func (d *DesktopHostIntegration) AutostartManaged() bool {
	return d.autostart != nil
}

func (d *DesktopHostIntegration) SetAutostart(enabled bool) error {
	if d.autostart == nil {
		return nil
	}
	var err error
	if enabled {
		err = d.autostart.Enable()
	} else {
		err = d.autostart.Disable()
	}
	if err == nil {
		if enabled {
			d.appendLog("已启用开机自启")
		} else {
			d.appendLog("已关闭开机自启")
		}
	}
	return err
}

func (d *DesktopHostIntegration) OpenControlCenter(_ string) error {
	d.mu.Lock()
	ctx := d.ctx
	d.mu.Unlock()
	if ctx == nil {
		return nil
	}
	wailsruntime.WindowShow(ctx)
	wailsruntime.WindowUnminimise(ctx)
	wailsruntime.WindowCenter(ctx)
	d.appendLog("已打开桌面控制台")
	return nil
}

func (d *DesktopHostIntegration) RecentLogs(limit int) []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if limit <= 0 || len(d.logs) == 0 {
		return nil
	}
	if limit > len(d.logs) {
		limit = len(d.logs)
	}
	result := make([]string, limit)
	copy(result, d.logs[:limit])
	return result
}

func (d *DesktopHostIntegration) UpdaterStatus() axis.UpdaterStatus {
	d.mu.Lock()
	updater := d.updater
	d.mu.Unlock()
	if updater == nil {
		return axis.UpdaterStatus{
			Mode:    "desktop-daemon",
			Running: false,
			State:   "idle",
			Message: "桌面更新服务未初始化",
		}
	}
	return updater.Status()
}

func (d *DesktopHostIntegration) CheckForUpdates() error {
	d.mu.Lock()
	updater := d.updater
	d.mu.Unlock()
	if updater == nil {
		return nil
	}
	if err := updater.CheckForUpdates(); err != nil {
		d.appendLog("桌面更新检查触发失败")
		return err
	}
	d.appendLog("已触发桌面更新检查")
	return nil
}

func (d *DesktopHostIntegration) Shutdown() error {
	d.mu.Lock()
	updater := d.updater
	d.mu.Unlock()
	if updater == nil {
		return nil
	}
	return updater.Stop()
}
