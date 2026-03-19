package app

import "github.com/TrueBlocks/trueblocks-acrylic/v2/internal/db"

func (a *App) GetProjects() ([]db.Project, error) {
	return a.db.GetProjects()
}

func (a *App) GetProject(id int) (db.Project, error) {
	return a.db.GetProject(id)
}

func (a *App) CreateProject(p db.Project) (int, error) {
	return a.db.CreateProject(p)
}

func (a *App) UpdateProject(p db.Project) error {
	return a.db.UpdateProject(p)
}

func (a *App) DeleteProject(id int) error {
	return a.db.DeleteProject(id)
}

func (a *App) GetProjectColorsWithMatches(projectID int) ([]db.ProjectColorWithMatches, error) {
	return a.db.GetProjectColorsWithMatches(projectID)
}

func (a *App) GetProjectColorCount(projectID int) (int, error) {
	return a.db.GetProjectColorCount(projectID)
}
