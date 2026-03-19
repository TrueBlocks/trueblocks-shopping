package main

import (
	"embed"
	"fmt"
	"log"

	appkit "github.com/TrueBlocks/trueblocks-art/packages/appkit/v2"
	"github.com/TrueBlocks/trueblocks-acrylic/v2/app"
	"github.com/TrueBlocks/trueblocks-acrylic/v2/internal/state"
	"github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	application := app.NewApp()
	stateManager := state.NewManager()

	err := appkit.Run(appkit.AppConfig{
		Title:             "Acrylic",
		Assets:            assets,
		Width:             1280,
		Height:            800,
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		GetWindowGeometry: stateManager.GetWindowGeometry,
		OnStartup:         application.Startup,
		OnShutdown:        application.Shutdown,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.trueblocks.acrylic.d4e7f2a1-9c3b-4e8d-b5a6-1f2e3d4c5b6a",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				_ = data
				fmt.Println("Cannot start a second instance")
			},
		},
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
