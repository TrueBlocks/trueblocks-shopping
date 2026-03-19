package db

import (
	"fmt"
	"time"
)

type Project struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	ImagePath       string `json:"imagePath"`
	ThumbnailPath   string `json:"thumbnailPath"`
	NColors         int    `json:"nColors"`
	TileSize        int    `json:"tileSize"`
	Posterize       bool   `json:"posterize"`
	SmoothingPasses int    `json:"smoothingPasses"`
	AspectRatio     string `json:"aspectRatio"`
	MatchOwnedOnly  bool   `json:"matchOwnedOnly"`
	Notes           string `json:"notes"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type ProjectColor struct {
	ID         int    `json:"id"`
	ProjectID  int    `json:"projectId"`
	SortOrder  int    `json:"sortOrder"`
	R          int    `json:"r"`
	G          int    `json:"g"`
	B          int    `json:"b"`
	Hex        string `json:"hex"`
	PixelCount int    `json:"pixelCount"`
}

type ColorMatch struct {
	ID          int         `json:"id"`
	ColorID     int         `json:"colorId"`
	MatchType   string      `json:"matchType"`
	Rank        int         `json:"rank"`
	DeltaE      float64     `json:"deltaE"`
	MatchRating string      `json:"matchRating"`
	Parts       []MatchPart `json:"parts"`
}

type MatchPart struct {
	ID      int    `json:"id"`
	MatchID int    `json:"matchId"`
	PaintID string `json:"paintId"`
	Parts   int    `json:"parts"`
	Paint   Paint  `json:"paint"`
}

type ProjectColorWithMatches struct {
	Color   ProjectColor `json:"color"`
	Matches []ColorMatch `json:"matches"`
}

func (db *DB) GetProjects() ([]Project, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, image_path, thumbnail_path, n_colors, tile_size,
			posterize, smoothing_passes, aspect_ratio, match_owned_only,
			notes, created_at, updated_at
		FROM projects ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (db *DB) GetProject(id int) (Project, error) {
	row := db.conn.QueryRow(
		`SELECT id, name, image_path, thumbnail_path, n_colors, tile_size,
			posterize, smoothing_passes, aspect_ratio, match_owned_only,
			notes, created_at, updated_at
		FROM projects WHERE id = ?`, id)

	var p Project
	var posterize, matchOwned int
	err := row.Scan(&p.ID, &p.Name, &p.ImagePath, &p.ThumbnailPath,
		&p.NColors, &p.TileSize, &posterize, &p.SmoothingPasses,
		&p.AspectRatio, &matchOwned, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, fmt.Errorf("get project %d: %w", id, err)
	}
	p.Posterize = posterize == 1
	p.MatchOwnedOnly = matchOwned == 1
	return p, nil
}

func (db *DB) CreateProject(p Project) (int, error) {
	posterize := 0
	if p.Posterize {
		posterize = 1
	}
	matchOwned := 0
	if p.MatchOwnedOnly {
		matchOwned = 1
	}

	result, err := db.conn.Exec(
		`INSERT INTO projects
			(name, image_path, thumbnail_path, n_colors, tile_size,
			posterize, smoothing_passes, aspect_ratio, match_owned_only, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.ImagePath, p.ThumbnailPath, p.NColors, p.TileSize,
		posterize, p.SmoothingPasses, p.AspectRatio, matchOwned, p.Notes)
	if err != nil {
		return 0, fmt.Errorf("create project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}
	return int(id), nil
}

func (db *DB) UpdateProject(p Project) error {
	posterize := 0
	if p.Posterize {
		posterize = 1
	}
	matchOwned := 0
	if p.MatchOwnedOnly {
		matchOwned = 1
	}

	_, err := db.conn.Exec(
		`UPDATE projects SET name=?, image_path=?, thumbnail_path=?,
			n_colors=?, tile_size=?, posterize=?,
			smoothing_passes=?, aspect_ratio=?, match_owned_only=?, notes=?,
			updated_at=?
		WHERE id=?`,
		p.Name, p.ImagePath, p.ThumbnailPath,
		p.NColors, p.TileSize, posterize, p.SmoothingPasses,
		p.AspectRatio, matchOwned, p.Notes,
		time.Now().UTC().Format("2006-01-02 15:04:05"), p.ID)
	return err
}

func (db *DB) DeleteProject(id int) error {
	_, err := db.conn.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

func (db *DB) GetProjectColors(projectID int) ([]ProjectColor, error) {
	rows, err := db.conn.Query(
		`SELECT id, project_id, sort_order, r, g, b, hex, pixel_count
		FROM project_colors WHERE project_id = ? ORDER BY sort_order`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query project colors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []ProjectColor
	for rows.Next() {
		var c ProjectColor
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.SortOrder,
			&c.R, &c.G, &c.B, &c.Hex, &c.PixelCount); err != nil {
			return nil, fmt.Errorf("scan project color: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (db *DB) InsertProjectColor(c ProjectColor) (int, error) {
	result, err := db.conn.Exec(
		`INSERT INTO project_colors (project_id, sort_order, r, g, b, hex, pixel_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ProjectID, c.SortOrder, c.R, c.G, c.B, c.Hex, c.PixelCount)
	if err != nil {
		return 0, fmt.Errorf("insert project color: %w", err)
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (db *DB) ClearProjectColors(projectID int) error {
	_, err := db.conn.Exec("DELETE FROM project_colors WHERE project_id = ?", projectID)
	return err
}

func (db *DB) InsertColorMatch(m ColorMatch) (int, error) {
	result, err := db.conn.Exec(
		`INSERT INTO color_matches (color_id, match_type, rank, delta_e, match_rating)
		VALUES (?, ?, ?, ?, ?)`,
		m.ColorID, m.MatchType, m.Rank, m.DeltaE, m.MatchRating)
	if err != nil {
		return 0, fmt.Errorf("insert color match: %w", err)
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (db *DB) InsertMatchPart(mp MatchPart) error {
	_, err := db.conn.Exec(
		`INSERT INTO match_parts (match_id, paint_id, parts) VALUES (?, ?, ?)`,
		mp.MatchID, mp.PaintID, mp.Parts)
	return err
}

func (db *DB) GetProjectColorsWithMatches(projectID int) ([]ProjectColorWithMatches, error) {
	colors, err := db.GetProjectColors(projectID)
	if err != nil {
		return nil, err
	}

	var result []ProjectColorWithMatches
	for _, c := range colors {
		matches, err := db.getColorMatches(c.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, ProjectColorWithMatches{Color: c, Matches: matches})
	}
	return result, nil
}

func (db *DB) getColorMatches(colorID int) ([]ColorMatch, error) {
	rows, err := db.conn.Query(
		`SELECT id, color_id, match_type, rank, delta_e, match_rating
		FROM color_matches WHERE color_id = ? ORDER BY rank`, colorID)
	if err != nil {
		return nil, fmt.Errorf("query color matches: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var matches []ColorMatch
	for rows.Next() {
		var m ColorMatch
		if err := rows.Scan(&m.ID, &m.ColorID, &m.MatchType,
			&m.Rank, &m.DeltaE, &m.MatchRating); err != nil {
			return nil, fmt.Errorf("scan color match: %w", err)
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range matches {
		parts, err := db.getMatchParts(matches[i].ID)
		if err != nil {
			return nil, err
		}
		matches[i].Parts = parts
	}
	return matches, nil
}

func (db *DB) getMatchParts(matchID int) ([]MatchPart, error) {
	rows, err := db.conn.Query(
		`SELECT mp.id, mp.match_id, mp.paint_id, mp.parts,
			p.id, p.brand, p.name, p.series, p.opacity, p.pigments,
			p.r, p.g, p.b, p.hex, p.lab_l, p.lab_a, p.lab_b, p.owned
		FROM match_parts mp
		JOIN paints p ON p.id = mp.paint_id
		WHERE mp.match_id = ?`, matchID)
	if err != nil {
		return nil, fmt.Errorf("query match parts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var parts []MatchPart
	for rows.Next() {
		var mp MatchPart
		var owned int
		if err := rows.Scan(&mp.ID, &mp.MatchID, &mp.PaintID, &mp.Parts,
			&mp.Paint.ID, &mp.Paint.Brand, &mp.Paint.Name, &mp.Paint.Series,
			&mp.Paint.Opacity, &mp.Paint.Pigments,
			&mp.Paint.R, &mp.Paint.G, &mp.Paint.B, &mp.Paint.Hex,
			&mp.Paint.LabL, &mp.Paint.LabA, &mp.Paint.LabB, &owned); err != nil {
			return nil, fmt.Errorf("scan match part: %w", err)
		}
		mp.Paint.Owned = owned == 1
		parts = append(parts, mp)
	}
	return parts, rows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanProject(s scanner) (Project, error) {
	var p Project
	var posterize, matchOwned int
	err := s.Scan(&p.ID, &p.Name, &p.ImagePath, &p.ThumbnailPath,
		&p.NColors, &p.TileSize, &posterize, &p.SmoothingPasses,
		&p.AspectRatio, &matchOwned, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, fmt.Errorf("scan project: %w", err)
	}
	p.Posterize = posterize == 1
	p.MatchOwnedOnly = matchOwned == 1
	return p, nil
}

func (db *DB) GetProjectColorCount(projectID int) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM project_colors WHERE project_id = ?", projectID,
	).Scan(&count)
	return count, err
}
