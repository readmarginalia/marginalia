package server

import (
	"os"
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"marginalia/db"
)

const (
	dbPath = "data/marginalia.db"
)

func initializeDb(t *testing.T) *sql.DB {
	os.Remove(dbPath)
	var db, err = db.Open(dbPath)

	if err != nil {
		t.Error(err)
	}

	return db
}

func Test_handleRSS(t *testing.T) {
	db := initializeDb(t)
	defer db.Close()

	req, err := http.NewRequest("GET", "/rss", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlerFunc := handleRSS(db, "test suite")

	handlerFunc.ServeHTTP(rr, req)

	expected := http.StatusOK
	if status := rr.Code; status != expected {
		t.Errorf("status %d -> %d", expected, status)
	}
}

func Test_handleAdd(t *testing.T) {
	db := initializeDb(t)
	defer db.Close()

	data := struct {
		Url string
	}{Url: "https://www.freecodecamp.org/news/vim-language-and-motions-explained/"}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/recommend", &buf)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handleAdd(db).ServeHTTP(rr, req)

	expected := http.StatusCreated
	if status := rr.Code; status != expected {
		t.Errorf("status %d -> %d", expected, status)
	}
}
