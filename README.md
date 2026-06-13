```
__  __          ___  __ 
\ \/ /__  ___  / _ \/ / 
 \  / _ \/ _ \/ , _/ /__
 /_/\___/\___/_/|_/____/
 
```

A URL shortener with a REST API, CLI, and terminal UI dashboard — all in a single Go binary.

![Go Version](https://img.shields.io/badge/go-1.25-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Features

- **Short URL generation** — SHA-256 + base58, deterministic per user
- **Custom aliases** — override the generated short code
- **Analytics** — track clicks per link with recent visit details
- **Async tracking** — non-blocking click recording via goroutines
- **QR codes** — generate QR codes from the CLI
- **TUI dashboard** — full-screen terminal UI (bubbletea) with live polling, split panels, keyboard shortcuts
- **Pluggable storage** — Redis (production) or in-memory (dev/fallback)
- **API key auth** — simple header-based authentication
- **Rate limiting** — token bucket per-IP limiter
- **Docker** — single-container deployment with Redis

## Quick Start

```bash
git clone https://github.com/shx-dow/yoorl.git
cd yoorl

# Start server (auto-falls back to in-memory store if Redis unavailable)
go run ./cmd/yoorl server

# Create a short URL
go run ./cmd/yoorl create https://example.com

# Launch the TUI dashboard
go run ./cmd/yoorl tui
```

Or with Docker:

```bash
docker compose up
```

## Usage

```
yoorl server                    Start the HTTP server
yoorl create <url>              Create a short URL
yoorl create --alias mylink <url>  Create with custom alias
yoorl delete <short-url>        Delete a short URL
yoorl update <short-url> <url>  Update destination
yoorl analytics <short-url>     View click analytics
yoorl qr <short-url>            Print a QR code
yoorl tui                       Open TUI dashboard
```

### TUI Key Bindings

| Key | Action          |
|-----|-----------------|
| `n` | Create new URL  |
| `d` | Delete selected |
| `c` | Copy short URL  |
| `Enter` | View analytics |
| `q` / `Esc` | Quit       |

## Configuration

| Variable         | Description              | Default                    |
|------------------|--------------------------|----------------------------|
| `PORT`           | Server port              | `8080`                     |
| `BASE_URL`       | Public base URL          | `http://localhost:8080/`   |
| `REDIS_ADDR`     | Redis address            | `localhost:6379`           |
| `REDIS_PASSWORD` | Redis password           | (empty)                    |
| `API_KEYS`       | Auth keys (`key:user`)   | (none — auth disabled)     |
| `YOORL_BASE_URL` | API URL for CLI/TUI      | `http://localhost:8080`    |
| `YOORL_API_KEY`  | API key for CLI/TUI      | (none)                     |

## Architecture

```
cmd/yoorl/main.go          — Entry point (server + CLI + TUI)
handler/                   — HTTP handlers (CRUD, redirect, analytics)
store/                     — Store interface + Redis + in-memory implementation
shortener/                 — Short URL generation
internal/analytics/        — Async click tracker
internal/middleware/       — Auth, rate limit, CORS, request ID
internal/tui/              — Bubbletea TUI dashboard
```

## Development

```bash
go test ./...
```

## License

MIT License - see LICENSE file for details.