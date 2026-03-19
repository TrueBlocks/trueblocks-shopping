package app

import "github.com/TrueBlocks/trueblocks-acrylic/v2/internal/db"

func (a *App) GetPaints() ([]db.Paint, error) {
	return a.db.GetPaints()
}

func (a *App) GetPaint(id string) (db.Paint, error) {
	return a.db.GetPaint(id)
}

func (a *App) SetPaintOwned(id string, owned bool) error {
	return a.db.SetPaintOwned(id, owned)
}

func (a *App) GetOwnedPaints() ([]db.Paint, error) {
	return a.db.GetOwnedPaints()
}

func (a *App) GetPaintFilterOptions() (db.PaintFilterOptions, error) {
	return a.db.GetPaintFilterOptions()
}

func (a *App) GetPaintProjectCount(paintID string) (int, error) {
	return a.db.GetPaintProjectCount(paintID)
}
