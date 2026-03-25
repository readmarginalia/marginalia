# Marginalia

A self-hosted service for saving articles worth reading. Submit a URL, and Marginalia extracts the readable content, stores it in SQLite, and serves it as an RSS feed or a simple HTML list.

## Features

- **Reader extraction** — fetches a URL and pulls out the article text, title, byline, and site name using [go-readability](https://codeberg.org/readeck/go-readability)
- **RSS feed** — serves all saved articles as an RSS 2.0 feed with full content (`/rss`)
- **HTML list** — a minimal browsable page of all recommendations (`/list`)
- **Deduplication** — the same URL is only stored once
- **Token auth** — write endpoints require a `?token=` query parameter

## API

### Add a recommendation

```sh
curl -X POST 'https://marginalia.yourdomain.com/recommend?token=TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com/article"}'
```

Returns `201` with `{"id": 1, "title": "Article Title"}`.

### Delete a recommendation

```sh
curl -X DELETE 'https://marginalia.yourdomain.com/recommend/1?token=TOKEN'
```

Returns `204` on success, `404` if not found.

### RSS feed

```
GET /rss
```

Returns RSS 2.0 XML. Supports `If-None-Match` and `If-Modified-Since` for conditional requests.

### HTML page

```
GET /
```

Browsable list of all recommendations.

## Running

### With Docker Compose

1. Put your auth token in `secret_token.txt`
2. Run:

```sh
docker compose up -d
```

The service will be available on port 9595.

### Without Docker

```sh
export TOKEN="your-secret-token"
export DB_PATH="data/marginalia.db"  # optional, this is the default
export PORT="9595"                   # optional, this is the default
export THEME="terminal"              # optional, this is the default
go run .
```

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `TOKEN` | *(required)* | Auth token for write endpoints. Also read from `/run/secrets/token`. |
| `DB_PATH` | `data/marginalia.db` | Path to the SQLite database file |
| `PORT` | `9595` | HTTP listen port |
| `OWNER` | *(empty)* | Your name. Personalizes the page title and RSS feed (e.g. `OWNER=Filippos` → "Filippos' Marginalia"). |
| `THEME` | `terminal` | Visual theme for the HTML page. Options: `terminal`, `classic`, `modern`, `daily`, `raw`, `win`. |

## Bookmarklet

You can add a browser bookmarklet to quickly save the current page to Marginalia. Create a new bookmark and set the URL to:

```
javascript:(function(){fetch('https://marginalia.yourdomain.com/recommend?token=TOKEN',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({url:location.href})}).then(r=>r.text().then(t=>alert(r.ok?'✓ Recommended!\n'+t:'Error: '+r.status+'\n'+t))).catch(e=>alert('Failed: '+e.message))})();
```

Replace `marginalia.yourdomain.com` with your instance's hostname and `TOKEN` with your auth token.

## Apple Shortcut

Use this [Shortcut template](https://www.icloud.com/shortcuts/949e3162cbca41d1b7c8968a226b3be2) to save pages to Marginalia from the iOS/macOS share sheet. After installing, replace the URL and token with your own.

## Project structure

```
main.go          — entrypoint
server/          — HTTP routes and handlers
server/themes/   — CSS theme files (classic, terminal, modern, daily, raw, win)
db/              — SQLite schema, queries
extract/         — article extraction from URLs
feed/            — RSS 2.0 rendering
Dockerfile       — multi-stage build
docker-compose.yml
```
