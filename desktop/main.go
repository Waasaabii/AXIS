package main

import (
	"context"
	"log"
	"os"

	frontendassets "github.com/Waasaabii/AXIS/frontend"
	"github.com/Waasaabii/AXIS/internal/axis"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	maybeRunUpdaterRole()

	service, err := axis.NewService(configPath())
	if err != nil {
		log.Fatal(err)
	}

	host := NewDesktopHostIntegration()
	updater, err := NewUpdaterManager(service.RuntimeDir(), appVersion())
	if err != nil {
		log.Fatal(err)
	}
	if err := updater.Start(); err != nil {
		log.Fatal(err)
	}
	host.SetUpdater(updater)
	service.SetHostIntegration(host)

	engineBindings := NewEngineBindings(service)
	hostBindings := NewHostBindings(service)
	assetHandler := axis.NewServer(service)
	staticFS, _, err := frontendassets.StaticFS()
	if err != nil {
		log.Fatal(err)
	}
	if staticFS == nil {
		staticFS = os.DirFS(".")
	}

	err = wails.Run(&options.App{
		Title:             "AXIS",
		Width:             1360,
		Height:            920,
		MinWidth:          1100,
		MinHeight:         760,
		AssetServer:       &assetserver.Options{Assets: staticFS, Handler: assetHandler},
		HideWindowOnClose: false,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.waasaabii.axis",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				_ = host.OpenControlCenter("")
			},
		},
		Bind: []interface{}{
			engineBindings,
			hostBindings,
		},
		OnStartup: func(ctx context.Context) {
			host.SetContext(ctx)
			host.appendLog("桌面宿主已启动")
		},
		OnShutdown: func(context.Context) {
			host.appendLog("桌面宿主已退出")
			_ = host.Shutdown()
			_ = service.Close()
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func appVersion() string {
	if value := os.Getenv("AXIS_VERSION"); value != "" {
		return value
	}
	return "0.1.0"
}

func configPath() string {
	if value := os.Getenv("PROXYRELAY_CONFIG"); value != "" {
		resolvedPath, err := axis.EnsureConfigPath(value)
		if err != nil {
			log.Fatal(err)
		}
		return resolvedPath
	}
	resolvedPath, err := axis.EnsureConfigPath("")
	if err != nil {
		log.Fatal(err)
	}
	return resolvedPath
}
