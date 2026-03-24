# Marginalia

A self-hosted service for saving articles worth reading. Submit a URL, and Marginalia extracts the readable content, stores it in SQLite, and serves it as an RSS feed or a simple HTML list.

## Features

- **Reader extraction** — fetches a URL and pulls out the article text, title, byline, and site name using [go-readability](https://codeberg.org/readeck/go-readability)
- **RSS feed** — serves all saved articles as an RSS 2.0 feed with full content (`/rss`)
- **HTML list** — a minimal browsable page of all recommendations (`/list`)
- **Deduplication** — the same URL is only stored once
- **Token auth** — write endpoints require a `?token=` query parameter

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/recommend?token=TOKEN` | Yes | Save a URL. Body: `{"url": "..."}` |
| `DELETE` | `/recommend/{id}?token=TOKEN` | Yes | Delete a recommendation by ID |
| `GET` | `/rss` | No | RSS 2.0 feed of all articles |
| `GET` | `/` | No | HTML page listing all articles |

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
go run .
```

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `TOKEN` | *(required)* | Auth token for write endpoints. Also read from `/run/secrets/token`. |
| `DB_PATH` | `data/marginalia.db` | Path to the SQLite database file |
| `PORT` | `9595` | HTTP listen port |
| `OWNER` | *(empty)* | Your name. Personalizes the page title and RSS feed (e.g. `OWNER=Filippos` → "Filippos' Marginalia"). |

## Project structure

```
main.go          — entrypoint
server/          — HTTP routes and handlers
db/              — SQLite schema, queries
extract/         — article extraction from URLs
feed/            — RSS 2.0 rendering
Dockerfile       — multi-stage build
docker-compose.yml
```
