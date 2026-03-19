package db

import "fmt"

type Favorite struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Notes     string         `json:"notes"`
	R         int            `json:"r"`
	G         int            `json:"g"`
	B         int            `json:"b"`
	Hex       string         `json:"hex"`
	CreatedAt string         `json:"createdAt"`
	Parts     []FavoritePart `json:"parts"`
}

type FavoritePart struct {
	ID         int    `json:"id"`
	FavoriteID int    `json:"favoriteId"`
	PaintID    string `json:"paintId"`
	Parts      int    `json:"parts"`
	Paint      Paint  `json:"paint"`
}

func (db *DB) GetFavorites() ([]Favorite, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, notes, r, g, b, hex, created_at
		FROM favorites ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query favorites: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []Favorite
	for rows.Next() {
		var f Favorite
		if err := rows.Scan(&f.ID, &f.Name, &f.Notes, &f.R, &f.G, &f.B,
			&f.Hex, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan favorite: %w", err)
		}
		items = append(items, f)
	}
	return items, rows.Err()
}

func (db *DB) GetFavorite(id int) (Favorite, error) {
	var f Favorite
	err := db.conn.QueryRow(
		`SELECT id, name, notes, r, g, b, hex, created_at
		FROM favorites WHERE id = ?`, id,
	).Scan(&f.ID, &f.Name, &f.Notes, &f.R, &f.G, &f.B, &f.Hex, &f.CreatedAt)
	if err != nil {
		return f, fmt.Errorf("get favorite %d: %w", id, err)
	}

	parts, err := db.getFavoriteParts(f.ID)
	if err != nil {
		return f, err
	}
	f.Parts = parts
	return f, nil
}

func (db *DB) CreateFavorite(f Favorite) (int, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.Exec(
		`INSERT INTO favorites (name, notes, r, g, b, hex) VALUES (?, ?, ?, ?, ?, ?)`,
		f.Name, f.Notes, f.R, f.G, f.B, f.Hex)
	if err != nil {
		return 0, fmt.Errorf("insert favorite: %w", err)
	}

	favID, _ := result.LastInsertId()

	for _, p := range f.Parts {
		_, err = tx.Exec(
			`INSERT INTO favorite_parts (favorite_id, paint_id, parts) VALUES (?, ?, ?)`,
			favID, p.PaintID, p.Parts)
		if err != nil {
			return 0, fmt.Errorf("insert favorite part: %w", err)
		}
	}

	return int(favID), tx.Commit()
}

func (db *DB) UpdateFavorite(f Favorite) error {
	_, err := db.conn.Exec(
		`UPDATE favorites SET name=?, notes=? WHERE id=?`,
		f.Name, f.Notes, f.ID)
	return err
}

func (db *DB) DeleteFavorite(id int) error {
	_, err := db.conn.Exec("DELETE FROM favorites WHERE id = ?", id)
	return err
}

func (db *DB) getFavoriteParts(favoriteID int) ([]FavoritePart, error) {
	rows, err := db.conn.Query(
		`SELECT fp.id, fp.favorite_id, fp.paint_id, fp.parts,
			p.id, p.brand, p.name, p.series, p.opacity, p.pigments,
			p.r, p.g, p.b, p.hex, p.lab_l, p.lab_a, p.lab_b, p.owned
		FROM favorite_parts fp
		JOIN paints p ON p.id = fp.paint_id
		WHERE fp.favorite_id = ?`, favoriteID)
	if err != nil {
		return nil, fmt.Errorf("query favorite parts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var parts []FavoritePart
	for rows.Next() {
		var fp FavoritePart
		var owned int
		if err := rows.Scan(&fp.ID, &fp.FavoriteID, &fp.PaintID, &fp.Parts,
			&fp.Paint.ID, &fp.Paint.Brand, &fp.Paint.Name, &fp.Paint.Series,
			&fp.Paint.Opacity, &fp.Paint.Pigments,
			&fp.Paint.R, &fp.Paint.G, &fp.Paint.B, &fp.Paint.Hex,
			&fp.Paint.LabL, &fp.Paint.LabA, &fp.Paint.LabB, &owned); err != nil {
			return nil, fmt.Errorf("scan favorite part: %w", err)
		}
		fp.Paint.Owned = owned == 1
		parts = append(parts, fp)
	}
	return parts, rows.Err()
}
