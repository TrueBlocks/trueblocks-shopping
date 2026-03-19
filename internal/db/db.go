package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrueBlocks/trueblocks-art/packages/color"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var embeddedSchema embed.FS

type DB struct {
	conn *sql.DB
	path string
}

func New(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)

	if _, err = conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err = conn.Exec("PRAGMA journal_mode = WAL"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	return &DB{conn: conn, path: dbPath}, nil
}

func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

func (db *DB) Begin() (*sql.Tx, error) {
	return db.conn.Begin()
}

func (db *DB) IsInitialized() (bool, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='paints'",
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) InitSchema() error {
	schemaSQL, err := embeddedSchema.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read embedded schema: %w", err)
	}
	if _, err = db.conn.Exec(string(schemaSQL)); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}
	return nil
}

func (db *DB) SeedPaints() error {
	var count int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM paints").Scan(&count); err != nil {
		return fmt.Errorf("count paints: %w", err)
	}
	if count > 0 {
		return nil
	}

	paints, err := color.LoadPaints()
	if err != nil {
		return fmt.Errorf("load paints from CSV: %w", err)
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`INSERT INTO paints
		(id, brand, name, series, opacity, pigments, r, g, b, hex, lab_l, lab_a, lab_b)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, p := range paints {
		_, err = stmt.Exec(
			p.ID, p.Brand, p.Name, p.Series, p.Opacity, p.Pigments,
			p.RGB[0], p.RGB[1], p.RGB[2], p.Hex,
			p.Lab.L, p.Lab.A, p.Lab.B,
		)
		if err != nil {
			return fmt.Errorf("insert paint %s: %w", p.ID, err)
		}
	}

	return tx.Commit()
}
