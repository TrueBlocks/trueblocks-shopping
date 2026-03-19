package app

import (
	"context"
	"fmt"
	"path/filepath"

	appkit "github.com/TrueBlocks/trueblocks-art/packages/appkit/v2"
	"github.com/TrueBlocks/trueblocks-acrylic/v2/internal/db"
	"github.com/TrueBlocks/trueblocks-acrylic/v2/internal/server"
	"github.com/TrueBlocks/trueblocks-acrylic/v2/internal/state"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	state      *state.Manager
	db         *db.DB
	fileServer *server.FileServer
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.state = state.NewManager()

	dataDir := appkit.AppDirFor("acrylic")

	a.fileServer = server.New(dataDir)
	if _, err := a.fileServer.Start(); err != nil {
		runtime.LogErrorf(ctx, "File server failed to start: %v", err)
	}

	dbPath := filepath.Join(dataDir, "acrylic.db")
	database, err := db.New(dbPath)
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to open database: %v", err)
		return
	}
	a.db = database

	initialized, err := database.IsInitialized()
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to check database: %v", err)
		return
	}

	if !initialized {
		if err := database.InitSchema(); err != nil {
			runtime.LogErrorf(ctx, "Failed to init schema: %v", err)
			return
		}
	}

	if err := database.SeedPaints(); err != nil {
		runtime.LogErrorf(ctx, "Failed to seed paints: %v", err)
	}
}

func (a *App) Shutdown(_ context.Context) {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			runtime.LogErrorf(a.ctx, "Failed to close database: %v", err)
		}
	}
}

func (a *App) GetImageURL(relativePath string) string {
	if a.fileServer == nil || a.fileServer.Port() == 0 {
		a.fileServer.Log("GetImageURL: file server not running, relativePath=%s", relativePath)
		return ""
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/images/%s", a.fileServer.Port(), relativePath)
	a.fileServer.Log("GetImageURL: %s -> %s", relativePath, url)
	return url
}
