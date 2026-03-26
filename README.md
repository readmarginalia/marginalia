# Marginalia

A self-hosted service for saving articles worth reading. Submit a URL, and Marginalia extracts the readable content, stores it in SQLite, and serves it as an RSS feed or a simple HTML list.

## Features

- **Reader extraction** — fetches a URL and pulls out the article text, title, byline, and site name using [go-readability](https://codeberg.org/readeck/go-readability)
- **RSS feed** — serves all saved articles as an RSS 2.0 feed with full content (`/rss`)
- **HTML list** — a minimal browsable page of all recommendations (`/`)
- **Deduplication** — the same URL is only stored once
- **Bearer auth** — write endpoints require an `Authorization: Bearer ...` header
- **Optional failed-auth throttling** — repeated invalid write attempts can be temporarily blocked per client

## API

### Add a recommendation

```sh
curl -X POST 'https://marginalia.yourdomain.com/recommend' \
	-H 'Authorization: Bearer TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com/article"}'
```

Returns `201` with `{"id": 1, "title": "Article Title"}`. Returns `409` if the URL has already been saved.

### Delete a recommendation

```sh
curl -X DELETE 'https://marginalia.yourdomain.com/recommend/1' \
  -H 'Authorization: Bearer TOKEN'
```

Returns `204` on success, `404` if not found.

Write endpoints return `429` if the client has been temporarily blocked due to repeated failed authentication (when `AUTH_RATE_LIMIT` is enabled).

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

An example environment file with all supported options is available at [.env.example](.env.example). The app does not load dotenv files automatically, so source it in your shell or have your process manager or Compose file provide the variables.

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `TOKEN` | *(required)* | Auth token for write endpoints. Also read from `/run/secrets/token`. |
| `DB_PATH` | `data/marginalia.db` | Path to the SQLite database file |
| `PORT` | `9595` | HTTP listen port |
| `OWNER` | *(empty)* | Your name. Personalizes the page title and RSS feed (e.g. `OWNER=Filippos` → "Filippos' Marginalia"). |
| `THEME` | `terminal` | Visual theme for the HTML page. Options: `terminal`, `classic`, `modern`, `daily`, `raw`, `win`. |
| `AUTH_RATE_LIMIT` | `false` | Enable failed-auth throttling on write endpoints. 5 failed attempts within 1 minute triggers a 10-minute lockout per client IP. Recommended for internet-exposed deployments. |
| `TRUST_PROXY` | `false` | Trust proxy-provided client IP headers for auth throttling and logging. Leave disabled unless the app sits behind a proxy you control. |
| `REAL_IP_HEADERS` | `CF-Connecting-IP,True-Client-IP,X-Real-IP,X-Forwarded-For` | Comma-separated header priority used when `TRUST_PROXY=true`. |
| `TRUSTED_PROXIES` | *(empty)* | Optional comma-separated proxy IPs or CIDRs. When set, forwarded client IP headers are only trusted if the immediate peer matches one of these ranges. |

## Proxy deployments

Failed-auth throttling is disabled by default. Enable it explicitly when you want repeated invalid write attempts to trigger temporary lockouts:

```sh
export AUTH_RATE_LIMIT=true
```

If the service sits behind nginx, Caddy, Traefik, Cloudflare, or a Worker-to-origin hop, enable proxy trust explicitly:

```sh
export TRUST_PROXY=true
export TRUSTED_PROXIES="203.0.113.10,203.0.113.0/24"
# optional: override header priority if your proxy uses something custom
export REAL_IP_HEADERS="CF-Connecting-IP,X-Forwarded-For"
```

When `TRUST_PROXY=true`, Marginalia checks the configured real-IP headers in order and uses the first valid IP for auth throttling and denial logs. If `TRUSTED_PROXIES` is empty, any immediate peer is treated as trusted once proxy mode is enabled.

If you already have a proxy or edge in front of the app, keep coarse rate limiting there as the first barrier and let Marginalia's built-in throttling act as a second barrier.

## Bookmarklet

You can add a browser bookmarklet to quickly save the current page to Marginalia. Create a new bookmark and set the URL to:

```
javascript:(function(){fetch('https://marginalia.yourdomain.com/recommend',{method:'POST',headers:{'Authorization':'Bearer TOKEN','Content-Type':'application/json'},body:JSON.stringify({url:location.href})}).then(r=>r.text().then(t=>alert(r.ok?'✓ Recommended!\n'+t:'Error: '+r.status+'\n'+t))).catch(e=>alert('Failed: '+e.message))})();
```

Replace `marginalia.yourdomain.com` with your instance's hostname and `TOKEN` with your auth token.

This bookmarklet sends an `Authorization` header cross-origin. The built-in CORS middleware allows it. If your reverse proxy overrides CORS headers, make sure it also includes `Authorization` in `Access-Control-Allow-Headers`.

## Apple Shortcut

Use this [Shortcut template](https://www.icloud.com/shortcuts/949e3162cbca41d1b7c8968a226b3be2) to save pages to Marginalia from the iOS/macOS share sheet. After installing, replace the URL and set the request's `Authorization` header to `Bearer TOKEN`.

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
