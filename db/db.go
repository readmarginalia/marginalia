package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Recommendation struct {
	ID       int64  `json:"id"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Byline   string `json:"byline"`
	Excerpt  string `json:"excerpt"`
	Content  string `json:"content"`
	SiteName string `json:"site_name"`
	AddedAt  int64  `json:"added_at"`
}

func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS recommendations (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		url       TEXT NOT NULL UNIQUE,
		title     TEXT,
		byline    TEXT,
		excerpt   TEXT,
		content   TEXT,
		site_name TEXT,
		added_at  INTEGER NOT NULL DEFAULT (unixepoch())
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	// migrate: add content column if missing
	db.Exec(`ALTER TABLE recommendations ADD COLUMN content TEXT`)

	return db, nil
}

func Insert(db *sql.DB, url, title, byline, excerpt, content, siteName string) (int64, bool, error) {
	res, err := db.Exec(
		`INSERT INTO recommendations (url, title, byline, excerpt, content, site_name) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(url) DO NOTHING`,
		url, title, byline, excerpt, content, siteName,
	)
	if err != nil {
		return 0, false, err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return 0, false, nil // duplicate
	}
	id, _ := res.LastInsertId()
	return id, true, nil
}

func Delete(db *sql.DB, id int64) (bool, error) {
	res, err := db.Exec(`DELETE FROM recommendations WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

func All(db *sql.DB) ([]Recommendation, error) {
	rows, err := db.Query(`SELECT id, url, title, byline, excerpt, content, site_name, added_at FROM recommendations ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []Recommendation
	for rows.Next() {
		var r Recommendation
		if err := rows.Scan(&r.ID, &r.URL, &r.Title, &r.Byline, &r.Excerpt, &r.Content, &r.SiteName, &r.AddedAt); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}
