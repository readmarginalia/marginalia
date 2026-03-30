package server

import (
	"crypto/sha256"
	"database/sql"
	_ "embed"
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
	"marginalia/wayback"
)

//go:embed images/building.columns.fill.svg
var cacheIcon string

func ownerTitle(owner string) string {
	if owner == "" {
		return "Marginalia"
	}
	if owner[len(owner)-1] == 's' || owner[len(owner)-1] == 'S' {
		return owner + "' Marginalia"
	}
	return owner + "'s Marginalia"
}

func New(database *sql.DB, auth AuthConfig, owner string, theme string) http.Handler {
	auth = auth.withDefaults()
	var limiter *failedAuthLimiter
	if auth.EnableRateLimit {
		limiter = newFailedAuthLimiter(defaultAuthFailureLimit, defaultAuthFailureWindow, defaultAuthBlockDuration)
	}

	title := ownerTitle(owner)
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(tokenAuth(auth, limiter))
		r.Post("/recommend", handleAdd(database))
		r.Delete("/recommend/{id}", handleDelete(database))
	})

	r.Get("/rss", handleRSS(database, owner, title))
	r.Get("/", handleList(database, title, theme))

	return r
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

		wayback.RequestSave(body.URL)

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

func handleRSS(database *sql.DB, owner, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recs, err := db.All(database)
		if err != nil {
			jsonError(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
			return
		}

		data, err := feed.Render(recs, owner)
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
<title>{{.Title}}</title>
<link rel="alternate" type="application/rss+xml" title="{{.Title}}" href="/rss">
<style>
{{.Style}}
</style>
</head>
<body>
<header>
  <h1>{{.Title}}</h1>
  <a class="rss-link" href="/rss" title="RSS Feed"><svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 256 256" fill="currentColor"><circle cx="68" cy="189" r="28"/><path d="M160 213h-34a89 89 0 0 0-89-89V90a123 123 0 0 1 123 123z"/><path d="M224 213h-34a157 157 0 0 0-157-157V22a191 191 0 0 1 191 191z"/></svg> rss</a>
</header>
<hr>
<ul>
{{range .Items}}<li>
  <a href="{{.URL}}">{{.Title}}</a>
  <div class="meta">{{if .Byline}}{{.Byline}}{{end}}{{if and .Byline .SiteName}} · {{end}}{{.SiteName}}{{if or .Byline .SiteName}} · {{end}}{{.AddedAtFmt}} · <a href="{{.CacheURL}}" target="_blank" rel="noopener noreferrer" title="Cached snapshot"><span style="display:inline-flex;align-items:center;width:12px;height:12px;vertical-align:middle">{{.CacheIcon}}</span></a></div>
</li>
{{else}}<li class="empty">Nothing here yet.</li>
{{end}}</ul>
<footer>
  <a href="/rss">rss feed</a>
</footer>
</body>
</html>`))

type listPage struct {
	Title string
	Style template.CSS
	Items []listItem
}

type listItem struct {
	URL        string
	Title      string
	Byline     string
	SiteName   string
	AddedAtFmt string
	CacheURL   string
	CacheIcon  template.HTML
}

func handleList(database *sql.DB, title string, style string) http.HandlerFunc {
	css := template.CSS(style)
	return func(w http.ResponseWriter, r *http.Request) {
		recs, err := db.All(database)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		items := make([]listItem, len(recs))
		for i, r := range recs {
			addedAt := time.Unix(r.AddedAt, 0).UTC()
			items[i] = listItem{
				URL:        r.URL,
				Title:      r.Title,
				Byline:     r.Byline,
				SiteName:   r.SiteName,
				CacheURL:   wayback.URL(addedAt, r.URL),
				CacheIcon:  template.HTML(cacheIcon),
				AddedAtFmt: addedAt.Format("Jan 2, 2006"),
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		listTmpl.Execute(w, listPage{Title: title, Style: css, Items: items})
	}
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
