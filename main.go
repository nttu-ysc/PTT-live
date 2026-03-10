package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"ptt-live/pttclient"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()
	pttClient := pttclient.NewPttClient()
	pttClient.Connect()
	defer pttClient.Close()

	// Build menu with About item
	appMenu := menu.NewMenu()
	appMenuItem := appMenu.AddSubmenu("PTT Live")
	appMenuItem.AddText("關於 PTT Live", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		runtime.EventsEmit(app.ctx, "show-about")
	})
	appMenuItem.AddSeparator()
	appMenuItem.AddText("檢查更新", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(app.ctx, "check-update-menu")
	})
	appMenuItem.AddSeparator()
	appMenuItem.AddText(fmt.Sprintf("版本 v%s", AppVersion), nil, nil)

	// Create application with options
	err := wails.Run(&options.App{
		Title:      "PTT Live",
		Width:      430,
		Height:     560,
		Fullscreen: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 80},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			pttClient.StartUp(ctx)
		},
		AlwaysOnTop: true,
		Bind: []interface{}{
			app,
			pttClient,
		},
		Menu:      appMenu,
		Frameless: false,
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       false,
			},
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               "",
			Theme:                             windows.SystemDefault,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:   windows.RGB(20, 20, 20),
				DarkModeTitleText:  windows.RGB(200, 200, 200),
				DarkModeBorder:     windows.RGB(20, 0, 20),
				LightModeTitleBar:  windows.RGB(200, 200, 200),
				LightModeTitleText: windows.RGB(20, 20, 20),
				LightModeBorder:    windows.RGB(200, 200, 200),
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
