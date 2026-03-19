package db

import "fmt"

type Paint struct {
	ID       string  `json:"id"`
	Brand    string  `json:"brand"`
	Name     string  `json:"name"`
	Series   int     `json:"series"`
	Opacity  string  `json:"opacity"`
	Pigments string  `json:"pigments"`
	R        int     `json:"r"`
	G        int     `json:"g"`
	B        int     `json:"b"`
	Hex      string  `json:"hex"`
	LabL     float64 `json:"labL"`
	LabA     float64 `json:"labA"`
	LabB     float64 `json:"labB"`
	Owned    bool    `json:"owned"`
}

type PaintFilterOptions struct {
	Brands    []string `json:"brands"`
	Opacities []string `json:"opacities"`
}

func (db *DB) GetPaints() ([]Paint, error) {
	rows, err := db.conn.Query(
		`SELECT id, brand, name, series, opacity, pigments,
			r, g, b, hex, lab_l, lab_a, lab_b, owned
		FROM paints ORDER BY brand, name`)
	if err != nil {
		return nil, fmt.Errorf("query paints: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []Paint
	for rows.Next() {
		var p Paint
		var owned int
		if err := rows.Scan(&p.ID, &p.Brand, &p.Name, &p.Series, &p.Opacity,
			&p.Pigments, &p.R, &p.G, &p.B, &p.Hex,
			&p.LabL, &p.LabA, &p.LabB, &owned); err != nil {
			return nil, fmt.Errorf("scan paint: %w", err)
		}
		p.Owned = owned == 1
		items = append(items, p)
	}
	return items, rows.Err()
}

func (db *DB) GetPaint(id string) (Paint, error) {
	var p Paint
	var owned int
	err := db.conn.QueryRow(
		`SELECT id, brand, name, series, opacity, pigments,
			r, g, b, hex, lab_l, lab_a, lab_b, owned
		FROM paints WHERE id = ?`, id,
	).Scan(&p.ID, &p.Brand, &p.Name, &p.Series, &p.Opacity,
		&p.Pigments, &p.R, &p.G, &p.B, &p.Hex,
		&p.LabL, &p.LabA, &p.LabB, &owned)
	if err != nil {
		return p, fmt.Errorf("get paint %s: %w", id, err)
	}
	p.Owned = owned == 1
	return p, nil
}

func (db *DB) SetPaintOwned(id string, owned bool) error {
	val := 0
	if owned {
		val = 1
	}
	_, err := db.conn.Exec("UPDATE paints SET owned = ? WHERE id = ?", val, id)
	return err
}

func (db *DB) GetOwnedPaints() ([]Paint, error) {
	rows, err := db.conn.Query(
		`SELECT id, brand, name, series, opacity, pigments,
			r, g, b, hex, lab_l, lab_a, lab_b, owned
		FROM paints WHERE owned = 1 ORDER BY brand, name`)
	if err != nil {
		return nil, fmt.Errorf("query owned paints: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []Paint
	for rows.Next() {
		var p Paint
		var owned int
		if err := rows.Scan(&p.ID, &p.Brand, &p.Name, &p.Series, &p.Opacity,
			&p.Pigments, &p.R, &p.G, &p.B, &p.Hex,
			&p.LabL, &p.LabA, &p.LabB, &owned); err != nil {
			return nil, fmt.Errorf("scan paint: %w", err)
		}
		p.Owned = owned == 1
		items = append(items, p)
	}
	return items, rows.Err()
}

func (db *DB) GetPaintFilterOptions() (PaintFilterOptions, error) {
	opts := PaintFilterOptions{}

	rows, err := db.conn.Query("SELECT DISTINCT brand FROM paints WHERE brand != '' ORDER BY brand")
	if err != nil {
		return opts, fmt.Errorf("query brands: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return opts, err
		}
		opts.Brands = append(opts.Brands, v)
	}

	rows2, err := db.conn.Query("SELECT DISTINCT opacity FROM paints WHERE opacity != '' ORDER BY opacity")
	if err != nil {
		return opts, fmt.Errorf("query opacities: %w", err)
	}
	defer func() { _ = rows2.Close() }()
	for rows2.Next() {
		var v string
		if err := rows2.Scan(&v); err != nil {
			return opts, err
		}
		opts.Opacities = append(opts.Opacities, v)
	}

	return opts, nil
}

func (db *DB) GetPaintProjectCount(paintID string) (int, error) {
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(DISTINCT pc.project_id)
		FROM match_parts mp
		JOIN color_matches cm ON cm.id = mp.match_id
		JOIN project_colors pc ON pc.id = cm.color_id
		WHERE mp.paint_id = ?`, paintID,
	).Scan(&count)
	return count, err
}
