package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// version 由发布构建通过 -ldflags "-X main.version=<tag>" 注入，本地开发保持 dev。
var version = "dev"

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "WorkBuddy 自动签到",
		Width:             820,
		Height:            640,
		MinWidth:          720,
		MinHeight:         520,
		StartHidden:       hiddenFromArgs(os.Args[1:]),
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 248, G: 250, B: 252, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "workbuddy-checkin-single-instance",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				app.showWindow()
			},
		},
		OnStartup:     app.startup,
		OnShutdown:    app.shutdown,
		OnBeforeClose: app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
