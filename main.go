package main

import (
	"context"
	"embed"

	"github.com/TrueBlocks/trueblocks-shopping/internal/settings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Load settings for initial window position/size
	savedSettings, err := settings.Load()
	if err != nil {
		savedSettings = &settings.Settings{
			WindowWidth:  1280,
			WindowHeight: 800,
		}
	}

	// Use saved dimensions or defaults
	width := savedSettings.WindowWidth
	height := savedSettings.WindowHeight
	if width < 800 {
		width = 1280
	}
	if height < 600 {
		height = 800
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:       "AcrylicMaster - Image Color Palette & Paint Matcher",
		Width:       width,
		Height:      height,
		StartHidden: true, // Hide until DOM is ready to avoid flash of empty state
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
		},
		OnDomReady: func(ctx context.Context) {
			app.domReady(ctx)

			// Set initial window position if saved
			if savedSettings.WindowX != 0 || savedSettings.WindowY != 0 {
				runtime.WindowSetPosition(ctx, savedSettings.WindowX, savedSettings.WindowY)
			}

			// Show window now that DOM is ready
			runtime.WindowShow(ctx)
		},
		OnBeforeClose: func(ctx context.Context) bool {
			// Save window position and size before closing
			x, y := runtime.WindowGetPosition(ctx)
			w, h := runtime.WindowGetSize(ctx)
			app.SaveWindowSettings(x, y, w, h)
			return false // Allow close
		},
		Bind: []interface{}{
			app,
		},
		// Drag and Drop Configuration
		// EnableFileDrop enables the internal mechanism to capture file paths dropped on the window
		// DisableWebViewDrop prevents the WebView from "opening" the image (navigating away from the React app)
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},
		// Platform-specific options
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarDefault(),
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
