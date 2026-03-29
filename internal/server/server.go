package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"marginalia/internal/auth"
	"marginalia/internal/feed"
	"marginalia/internal/infra/http"
	"marginalia/internal/recommendations"
	stdhttp "net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type App struct {
	AuthConfig      *auth.AuthConfig
	Database        *sql.DB
	Owner           string
	Theme           string
	Feed            *feed.Service
	Recommendations *recommendations.Service
}

func ownerTitle(owner string) string {
	if owner == "" {
		return "Marginalia"
	}
	if owner[len(owner)-1] == 's' || owner[len(owner)-1] == 'S' {
		return owner + "' Marginalia"
	}
	return owner + "'s Marginalia"
}

func New(app *App) stdhttp.Handler {
	authConfig := app.AuthConfig.WithDefaults()
	var limiter *http.FailedAuthLimiter
	if authConfig.EnableRateLimit {
		limiter = http.DefaultFailedAuthLimiter()
	}

	title := ownerTitle(app.Owner)
	r := chi.NewRouter()

	r.Use(func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == stdhttp.MethodOptions {
				w.WriteHeader(stdhttp.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.TokenAuth(authConfig, limiter))
		r.Post("/recommend", handleAdd(app))
		r.Delete("/recommend/{id}", handleDelete(app))
	})

	r.Get("/rss", handleRSS(app))
	r.Get("/", handleList(app, title, app.Theme))

	return r
}

func handleAdd(app *App) stdhttp.HandlerFunc {
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		var body struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
			http.JsonError(w, "missing or invalid url", stdhttp.StatusBadRequest)
			return
		}

		rec, err := app.Recommendations.Insert(&recommendations.CreateOptions{URL: body.URL})
		if err != nil {
			http.WriteError(w, err)
			return
		}

		slog.Info("added recommendation", "url", body.URL, "title", rec.Title)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(stdhttp.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": rec.ID, "title": rec.Title})
	}
}

func handleDelete(app *App) stdhttp.HandlerFunc {
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.JsonError(w, "invalid id", stdhttp.StatusBadRequest)
			return
		}

		if err := app.Recommendations.Delete(id); err != nil {
			http.WriteError(w, err)
			return
		}
		w.WriteHeader(stdhttp.StatusNoContent)
	}
}

func handleRSS(app *App) stdhttp.HandlerFunc {
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		result, err := app.Feed.RenderRss(app.Owner)
		if err != nil {
			http.JsonError(w, fmt.Sprintf("feed error: %v", err), stdhttp.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/rss+xml")
		w.Header().Set("ETag", result.ETag)
		w.Header().Set("Last-Modified", result.LastModified.Format(stdhttp.TimeFormat))
		w.Header().Set("Cache-Control", "no-store, must-revalidate")

		slog.Info("rss request",
			"method", r.Method,
			"url", r.URL.String(),
			"If-None-Match", r.Header.Get("If-None-Match"),
			"If-Modified-Since", r.Header.Get("If-Modified-Since"))

		if match := r.Header.Get("If-None-Match"); match == result.ETag {
			w.WriteHeader(stdhttp.StatusNotModified)
			return
		}
		if ims := r.Header.Get("If-Modified-Since"); ims != "" {
			if t, err := stdhttp.ParseTime(ims); err == nil && !result.LastModified.After(t) {
				w.WriteHeader(stdhttp.StatusNotModified)
				return
			}
		}

		w.Write(result.Content)
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
  <div class="meta">{{if .Byline}}{{.Byline}}{{end}}{{if and .Byline .SiteName}} · {{end}}{{.SiteName}}{{if or .Byline .SiteName}} · {{end}}{{.AddedAtFmt}}</div>
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
}

func handleList(app *App, title string, style string) stdhttp.HandlerFunc {
	css := template.CSS(style)
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		recs, err := app.Recommendations.All()
		if err != nil {
			http.JsonError(w, err.Error(), stdhttp.StatusInternalServerError)
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
		listTmpl.Execute(w, listPage{Title: title, Style: css, Items: items})
	}
}
