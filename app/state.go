package app

import (
	appkit "github.com/TrueBlocks/trueblocks-art/packages/appkit/v2"
	"github.com/TrueBlocks/trueblocks-acrylic/v2/internal/state"
)

func (a *App) GetSidebarWidth() int                       { return a.state.GetSidebarWidth() }
func (a *App) SetSidebarWidth(width int)                  { a.state.SetSidebarWidth(width) }
func (a *App) GetTableState(n string) state.TableState    { return a.state.GetTableState(n) }
func (a *App) SetTableState(n string, t state.TableState) { a.state.SetTableState(n, t) }
func (a *App) GetTab(pageName string) string              { return a.state.GetTab(pageName) }
func (a *App) SetTab(pageName, tab string)                { a.state.SetTab(pageName, tab) }
func (a *App) SaveWindowGeometry(x, y, w, h int)          { a.state.SetWindowGeometry(x, y, w, h) }
func (a *App) GetTabRoute(key string) string              { return a.state.GetTabRoute(key) }
func (a *App) SetTabRoute(key, route string)              { a.state.SetTabRoute(key, route) }
func (a *App) GetLastRoute() string                       { return a.state.GetLastRoute() }
func (a *App) SaveLastRoute(route string)                 { a.state.SetLastRoute(route) }
func (a *App) GetThemeState() appkit.UIPreferences        { return a.state.GetThemeState() }
func (a *App) SaveThemeState(prefs appkit.UIPreferences)  { a.state.SetThemeState(prefs) }
