package db

import (
	"fmt"
	"database/sql"
	"path"
	"testing"
)

const (
	dbPath = "data/marginalia.db"
)

func initializeDb(t *testing.T) *sql.DB {
	dbPath := path.Join(t.TempDir(), dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestOpenRecommendations(t *testing.T) {
	const table = "recommendations"

	db := initializeDb(t)

	assertTableExists(t, db, "recommendations")

	for _, col := range []string{"id", "url", "title", "byline", "excerpt", "content", "site_name", "added_at"} {
		assertColumnExists(t, db, table, col)
	}
}

func assertColumnExists(t *testing.T, db *sql.DB, tableName, colName string) {
	t.Helper()

	var name string
	query := fmt.Sprintf("SELECT name FROM pragma_table_info('%s') WHERE name = ?;", tableName)
	
	err := db.QueryRow(query, colName).Scan(&name)
	if err == sql.ErrNoRows {
		t.Errorf("column %s not found in table %s", colName, tableName)
		return
	}
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()

	var name string
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?;`
	err := db.QueryRow(query, tableName).Scan(&name)

	if err == sql.ErrNoRows {
		t.Errorf("table %s does not exist", tableName)
	} else if err != nil {
		t.Errorf("querying sqlite_master: %v", err)
	}
}
