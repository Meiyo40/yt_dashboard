# AGENTS.md — YouTube Comment Dashboard

## Project Overview
Go + Templ + HTMX + Alpine.js dashboard that tracks replies to your own YouTube comments and lets you reply back from a custom UI.

## Tech Stack
| Layer | Technology |
|---|---|
| Backend | Go (stdlib + chi) |
| Templates | `templ` (type-safe server-side HTML) |
| Interactivity | HTMX (partial HTML swaps) |
| Client state | Alpine.js (dropdowns, toggles only) |
| Styling | Tailwind CSS (CDN Play — no build step) |
| DB | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |

## Dev Commands
```bash
# One-time setup
go install github.com/a-h/templ/cmd/templ@latest
go install github.com/air-verse/air@latest

# Generate .templ -> .go files (required before every run)
templ generate

# Hot reload dev server (watches .go + .templ)
air

# Production build
go build -o ./bin/yt-dashboard .
```

## Project Structure (expected)
```
main.go              # Wiring only: routes, server config
handlers/
  pages.go           # Full-page handlers (GET /)
  partials.go        # HTMX partial handlers (GET /partials/*)
templates/
  layout.templ       # Base layout (head, nav, scripts)
  pages/             # Page templates
  partials/          # HTMX fragment templates
static/              # Static assets (if any)
```

## Architecture Constraints
- **No API endpoint exists to list all comments by a user** — the app must maintain a local Comment ID Registry (SQLite) of comment IDs the user has posted, then poll `comments.list?parentId={id}` for replies.
- **YouTube threading is flat** — all replies share the same `parentId` (the top-level comment). When replying, `parentId` is always the original comment ID, never a reply's ID.
- **Quota budget**: 10,000 units/day. `comments.list` = 1 unit, `comments.insert` = 50 units. Budget ~200 replies/day max.
- **OAuth scopes**: `youtube.readonly` for reads, `youtube.force-ssl` for writes. Tokens expire in 1h — use `refresh_token`.

## Conventions
- `main.go` = wiring only (routes, server init). No business logic.
- Handler flow: handler → data → template. No service/repository layer for this scale.
- One handler = one file, grouped by responsibility (pages vs partials).
- Templates mirror handler structure. No business logic in templates.
- Stable HTML IDs for HTMX targets (e.g., `id="table-wrapper"`).
- Alpine only for local UI state (dropdowns, tabs, modals). Never use `fetch()` in Alpine if HTMX can do it.
- `templ generate` must run before `go run` or `go build`.

## Error Handling
- Errors handled inline in handlers. No complex error middleware.
- YouTube API errors: 403 (comments disabled/video deleted/quota exceeded), 401 (token expired → refresh), 400 (bad request).
