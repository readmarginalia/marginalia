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

func ownerTitle(owner string) string {
	if owner == "" {
		return "Marginalia"
	}
	if owner[len(owner)-1] == 's' || owner[len(owner)-1] == 'S' {
		return owner + "' Marginalia"
	}
	return owner + "'s Marginalia"
}

func New(database *sql.DB, token string, owner string) http.Handler {
	title := ownerTitle(owner)
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(tokenAuth(token))
		r.Post("/recommend", handleAdd(database))
		r.Delete("/recommend/{id}", handleDelete(database))
	})

	r.Get("/rss", handleRSS(database, owner, title))
	r.Get("/", handleList(database, title))

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
  :root { color-scheme: light dark; }
  body {
    font-family: system-ui, sans-serif;
    font-size: 17px;
    line-height: 1.55;
    letter-spacing: -0.003em;
    font-kerning: normal;
    text-rendering: optimizeLegibility;
    max-width: 700px;
    margin: 2rem auto;
    padding: 0 1rem;
    background: #fff;
    color: #111;
  }
  header { display: flex; align-items: baseline; justify-content: space-between; }
  h1 { font-size: 1.6rem; line-height: 1.2; margin-bottom: 0.3em; }
  .rss-link { color: #666; font-size: 0.85em; text-decoration: none; font-weight: normal; display: inline-flex; align-items: center; gap: 0.3em; }
  .rss-link:hover { color: #1a4fd8; }
  ul { list-style: none; padding: 0; margin: 0; }
  li { margin-bottom: 0; padding: 0.85em 0; border-bottom: 1px solid #e3e3e3; }
  li:last-child { border-bottom: none; padding-bottom: 0; }
  a { color: #1a4fd8; text-decoration: none; font-weight: 600; }
  a:visited { color: #1a4fd8; }
  a:hover { color: #0f3fb5; }
  a:active { color: #0c3290; }
  .meta { color: #666; font-size: 0.85em; margin-top: 0.2em; }
  footer {
    margin-top: 1em;
    padding-top: 1em;
    border-top: 1px solid #e3e3e3;
    font-size: 0.85em;
    color: #666;
  }
  footer a, footer a:visited, footer a:hover, footer a:active {
    color: inherit;
    text-decoration: underline;
    text-underline-offset: 0.12em;
  }
  @media (prefers-color-scheme: dark) {
    body { background: #111; color: #eee; }
    li { border-bottom-color: #333; }
    .meta { color: #aaa; }
    a { color: #7aa2ff; }
    a:visited { color: #7aa2ff; }
    a:hover { color: #9db8ff; }
    a:active { color: #5f86e8; }
    footer { border-top-color: #333; color: #aaa; }
  }
  @media (max-width: 700px) {
    body { font-size: 18px; line-height: 1.56; }
  }
</style>
</head>
<body>
<header>
  <h1>{{.Title}}</h1>
  <a class="rss-link" href="/rss" title="RSS Feed"><svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 256 256" fill="currentColor"><circle cx="68" cy="189" r="28"/><path d="M160 213h-34a89 89 0 0 0-89-89V90a123 123 0 0 1 123 123z"/><path d="M224 213h-34a157 157 0 0 0-157-157V22a191 191 0 0 1 191 191z"/></svg> RSS</a>
</header>
<ul>
{{range .Items}}<li>
  <a href="{{.URL}}">{{.Title}}</a>
  <div class="meta">{{if .Byline}}by {{.Byline}}{{end}}{{if and .Byline .SiteName}} · {{end}}{{.SiteName}}{{if or .Byline .SiteName}} · {{end}}{{.AddedAtFmt}}</div>
</li>
{{else}}<li>No recommendations yet.</li>
{{end}}</ul>
<footer>
  <a href="/rss">RSS Feed</a>
</footer>
</body>
</html>`))

type listPage struct {
	Title string
	Items []listItem
}

type listItem struct {
	URL        string
	Title      string
	Byline     string
	SiteName   string
	AddedAtFmt string
}

func handleList(database *sql.DB, title string) http.HandlerFunc {
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
		listTmpl.Execute(w, listPage{Title: title, Items: items})
	}
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
