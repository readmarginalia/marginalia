package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/XSAM/otelsql"
	_ "modernc.org/sqlite"
)

func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	driverName, err := otelsql.Register("sqlite")
	if err != nil {
		return nil, fmt.Errorf("register otelsql driver: %w", err)
	}

	db, err := sql.Open(driverName, dbPath)
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

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS identity (
		id          INTEGER PRIMARY KEY CHECK (id = 1),
		public_key  TEXT NOT NULL,
		private_key TEXT NOT NULL,
		created_at  INTEGER NOT NULL DEFAULT (unixepoch())
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create identity table: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS peers (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		endpoint    TEXT NOT NULL UNIQUE,
		public_key  TEXT NOT NULL,
		owner       TEXT,
		status      TEXT NOT NULL DEFAULT 'discovered' CHECK (status IN ('trusted', 'discovered')),
		pinned_at   INTEGER,
		last_seen   INTEGER,
		added_at    INTEGER NOT NULL DEFAULT (unixepoch())
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create peers table: %w", err)
	}

	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_peers_status ON peers(status)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create peers index: %w", err)
	}

	return db, nil
}
