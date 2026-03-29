package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"marginalia/db"
)

const (
	dbPath = "data/marginalia.db"
)

func initializeDb(t *testing.T) *sql.DB {
	dbPath := path.Join(t.TempDir(), dbPath)

	db, err := db.Open(dbPath)
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func Test_handleRSS(t *testing.T) {
	db := initializeDb(t)

	req, err := http.NewRequest("GET", "/rss", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlerFunc := handleRSS(db, "test suite")
	handlerFunc.ServeHTTP(rr, req)

	expected := http.StatusOK
	if status := rr.Code; status != expected {
		t.Fatalf("expected status %d -> %d", expected, status)
	}
}

func Test_handleAdd(t *testing.T) {
	db := initializeDb(t)

	var tests = []struct {
		name           string
		url            string
		httpStatusCode int
	}{
		{"400 BadRequest on empty url", "", 400},
		{"201 OK on new url", "https://www.freecodecamp.org/news/vim-language-and-motions-explained/", 201},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := struct {
				Url string
			}{Url: tt.url}

			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(data); err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequest("POST", "/recommend", &buf)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			handleAdd(db).ServeHTTP(recorder, req)

			if status := recorder.Code; status != tt.httpStatusCode {
				t.Fatalf("expected status %d -> %d", tt.httpStatusCode, status)
			}
		})
	}
}
