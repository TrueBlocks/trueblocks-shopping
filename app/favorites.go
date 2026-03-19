package app

import "github.com/TrueBlocks/trueblocks-acrylic/v2/internal/db"

func (a *App) GetFavorites() ([]db.Favorite, error) {
	return a.db.GetFavorites()
}

func (a *App) GetFavorite(id int) (db.Favorite, error) {
	return a.db.GetFavorite(id)
}

func (a *App) CreateFavorite(f db.Favorite) (int, error) {
	return a.db.CreateFavorite(f)
}

func (a *App) UpdateFavorite(f db.Favorite) error {
	return a.db.UpdateFavorite(f)
}

func (a *App) DeleteFavorite(id int) error {
	return a.db.DeleteFavorite(id)
}
