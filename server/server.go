package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"marginalia/db"
	"marginalia/extract"
	"marginalia/feed"
)

func New(database *sql.DB, token string) http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(tokenAuth(token))
		r.Post("/recommend", handleAdd(database))
		r.Delete("/recommend/{id}", handleDelete(database))
	})

	r.Get("/rss", handleRSS(database))
	r.Get("/list", handleList(database))

	return r
}

func tokenAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("token") != token {
				jsonError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func handleAdd(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
			jsonError(w, "missing or invalid url", http.StatusBadRequest)
			return
		}

		article, err := extract.FromURL(body.URL)
		if err != nil {
			jsonError(w, fmt.Sprintf("extraction failed: %v", err), http.StatusBadGateway)
			return
		}

		id, inserted, err := db.Insert(database, body.URL, article.Title, article.Byline, article.Excerpt, article.Content, article.SiteName)
		if err != nil {
			jsonError(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
			return
		}
		if !inserted {
			jsonError(w, "url already exists", http.StatusConflict)
			return
		}

		log.Printf("added: %s — %s", body.URL, article.Title)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id, "title": article.Title})
	}
}

func handleDelete(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, "invalid id", http.StatusBadRequest)
			return
		}

		found, err := db.Delete(database, id)
		if err != nil {
			jsonError(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
			return
		}
		if !found {
			jsonError(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleRSS(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recs, err := db.All(database)
		if err != nil {
			jsonError(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
			return
		}

		data, err := feed.Render(recs)
		if err != nil {
			jsonError(w, fmt.Sprintf("feed error: %v", err), http.StatusInternalServerError)
			return
		}

		hash := sha256.Sum256(data)
		etag := `"` + hex.EncodeToString(hash[:8]) + `"`

		// Use the most recent item's timestamp as Last-Modified
		var lastMod time.Time
		if len(recs) > 0 {
			lastMod = time.Unix(recs[0].AddedAt, 0).UTC()
		} else {
			lastMod = time.Now().UTC()
		}

		w.Header().Set("Content-Type", "application/rss+xml")
		w.Header().Set("ETag", etag)
		w.Header().Set("Last-Modified", lastMod.Format(http.TimeFormat))
		w.Header().Set("Cache-Control", "no-store, must-revalidate")

		log.Printf("rss: %s %s If-None-Match=%q If-Modified-Since=%q",
			r.Method, r.URL, r.Header.Get("If-None-Match"), r.Header.Get("If-Modified-Since"))

		if match := r.Header.Get("If-None-Match"); match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if ims := r.Header.Get("If-Modified-Since"); ims != "" {
			if t, err := http.ParseTime(ims); err == nil && !lastMod.After(t) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Write(data)
	}
}

var listTmpl = template.Must(template.New("list").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Marginalia</title>
<style>
  body { font-family: system-ui, sans-serif; max-width: 700px; margin: 2rem auto; padding: 0 1rem; color: #222; }
  h1 { font-size: 1.5rem; }
  ul { list-style: none; padding: 0; }
  li { margin-bottom: 1.2rem; padding-bottom: 1.2rem; border-bottom: 1px solid #eee; }
  a { color: #1a0dab; text-decoration: none; font-weight: 600; }
  a:hover { text-decoration: underline; }
  .meta { color: #666; font-size: 0.85rem; margin-top: 0.2rem; }
</style>
</head>
<body>
<h1>Marginalia</h1>
<ul>
{{range .}}<li>
  <a href="{{.URL}}">{{.Title}}</a>
  <div class="meta">{{if .Byline}}by {{.Byline}}{{end}}{{if and .Byline .SiteName}} · {{end}}{{.SiteName}}{{if or .Byline .SiteName}} · {{end}}{{.AddedAtFmt}}</div>
</li>
{{else}}<li>No recommendations yet.</li>
{{end}}</ul>
</body>
</html>`))

type listItem struct {
	URL       string
	Title     string
	Byline    string
	SiteName  string
	AddedAtFmt string
}

func handleList(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recs, err := db.All(database)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		items := make([]listItem, len(recs))
		for i, r := range recs {
			items[i] = listItem{
				URL:        r.URL,
				Title:      r.Title,
				Byline:     r.Byline,
				SiteName:   r.SiteName,
				AddedAtFmt: time.Unix(r.AddedAt, 0).UTC().Format("Jan 2, 2006"),
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		listTmpl.Execute(w, items)
	}
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
