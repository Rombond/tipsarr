# TipsArr

A Jellyseerr-style dashboard that unifies three home-lab services into one login-gated app:

- **[BoxArr](#boxarr)** — weekly box office tracking against your Radarr library
- **[SuggestArr](#suggestarr)** — personalized suggestion jobs
- **[Seerr](#seerr)** (Jellyseerr/Overseerr) — discovery, requests, and user management

Log in once with your Jellyfin account and get one screen: a Discover page with infinite-scroll
carousels, full media/person/collection detail pages, a search bar, a requests list, and a minimal
admin area — instead of juggling three separate UIs.

Built with **SvelteKit (Svelte 5)**, **shadcn-svelte**, and **Tailwind CSS**, running on **Bun**.

## Screens

| Route | What it does |
|---|---|
| `/login` | Sign in with Jellyfin credentials. Authenticates against Seerr; auto-provisions and links a SuggestArr account on first login. |
| `/discover` | 5 infinite-scroll carousels: Box Office, Suggestions, Trending, Discover Movies, Discover TV. |
| `/media/[type]/[id]` | Full detail page for a movie/TV show — backdrop, cast, recommendations, collection banner, info panel, request button. |
| `/person/[id]` | Actor/crew detail page with bio and combined filmography. |
| `/collection/[id]` | All movies in a franchise/collection. |
| `/search` | Search movies, TV shows, and people across Seerr's catalog. |
| `/requests` | Your requests (or everyone's, if you're an admin). |
| `/admin/users`, `/admin/settings` | Admin-only: Seerr user list with roles, basic backend config display. |

## Quick Start

```bash
bun install
cp .env.example .env   # then fill in the URLs/keys below
bun dev                 # dev server (default: 0.0.0.0:5173)
```

```bash
bun dev                      # dev server
bun dev --open                # + opens browser
bun check                     # type-check without running
bun check --watch             # type-check with watch mode
bun build                     # production build
bun preview                   # preview production build
```

## Configuring the Backends

Copy `.env.example` to `.env` and set:

```bash
# BoxArr — no auth
VITE_BOXARR_URL=http://localhost:8080/api

# SuggestArr — JWT via httpOnly cookie. Requires ALLOW_REGISTRATION=true on the
# SuggestArr backend so TipsArr can auto-create + link an account on first login.
VITE_SUGGESTARR_URL=http://localhost:8080/api

# Radarr — LAN-only, ships an admin-capable API key in the built JS bundle.
VITE_RADARR_URL=http://localhost:7878
VITE_RADARR_API_KEY=

# TMDB — fallback poster lookup when Radarr has no match.
VITE_TMDB_API_URL=https://api.themoviedb.org/3
VITE_TMDB_IMAGE_BASE=https://image.tmdb.org/t/p/original
VITE_TMDB_API_KEY=

# Seerr (Jellyseerr/Overseerr) — session-cookie auth via Jellyfin login.
# Seerr must already have a Jellyfin server configured under Settings > Jellyfin.
VITE_SEERR_URL=http://localhost:5055
```

**Security note:** everything above ships into the built JS bundle and calls services directly
from the browser. This app is designed for **LAN-only / trusted-network deployment**, not for
exposing to the public internet as-is.

### Why the proxy in `vite.config.ts`?

Seerr doesn't send CORS headers, so the dev/preview server proxies `/seerr-api/*` → `VITE_SEERR_URL`
(see `vite.config.ts`). This also lets Seerr's session cookie ride along as same-origin. If you
change `VITE_SEERR_URL`, restart the dev/preview server — Vite only reads env vars for the proxy
target at config-load time.

## How Login Works

1. `POST /auth/jellyfin` against Seerr with the entered credentials — this is the real credential
   check and the primary gate. Seerr must already have Jellyfin configured as its media server.
2. In parallel, TipsArr tries to log into SuggestArr with the same credentials. If no SuggestArr
   account exists yet, it registers one and links it to the matching Jellyfin user automatically.
   This step is **best-effort**: if SuggestArr is unreachable or registration is disabled, login
   still succeeds and the Suggestions carousel just shows a "not connected" state instead.
3. Admin status is the union of Seerr's permission bitmask (`ADMIN` / `MANAGE_USERS`) and
   SuggestArr's `role: 'admin'` — either one grants admin nav items (`Users`, `Settings`).

Known limitation: SuggestArr's refresh cookie is `SameSite=Strict` and called cross-origin (not
proxied like Seerr), so that session doesn't survive a full page reload — a fresh login re-links it
each time.

## Architecture

```
src/
├── routes/
│   ├── +layout.svelte              # Auth guard + sidebar shell
│   ├── login/, discover/, requests/, search/
│   ├── media/[type]/[id]/          # Movie/TV detail page
│   ├── person/[id]/                # Actor/crew detail page
│   ├── collection/[id]/            # Franchise/collection page
│   └── admin/users/, admin/settings/
├── lib/
│   ├── api/
│   │   ├── boxarr.ts               # GET /boxoffice/current, /movies/{id}, /movies/status, /weeks
│   │   ├── suggestarr.ts           # Auth, suggestions, workflow approvals, user linking
│   │   ├── seerr.ts                # Auth, discover, requests, users, search, person, collection
│   │   ├── radarr.ts / tmdb.ts     # Poster lookups for BoxArr items
│   │   └── media.ts                # Normalizes all 3 sources into one `MediaItem` shape
│   ├── stores/auth.svelte.ts       # Unified session store (Seerr + SuggestArr identities)
│   └── components/
│       ├── layout/                 # Sidebar + app shell
│       ├── media/                  # Generic carousel, media card, unified detail modal
│       ├── discover/               # Per-source carousel wrappers
│       ├── requests/, admin/, auth/
│       └── ui/                     # shadcn-svelte primitives
└── app.html
```

The core idea: every carousel/card/modal works off one normalized `MediaItem` type
(`src/lib/api/media.ts`), regardless of whether the underlying data came from BoxArr, SuggestArr,
or Seerr. That's what lets one `Carousel` and one `MediaDetailModal` component serve all five
Discover rows.

## Backend References

- `docs/seerr/seerr-api.yml` — full Seerr (Jellyseerr/Overseerr) OpenAPI spec
- `docs/suggestarr/API.md` — SuggestArr backend API reference
- `docs/boxarr/openapi.json` — BoxArr OpenAPI spec

## File Conventions

- `*.svelte` — Svelte 5 components (`<script lang="ts">`, `$state()`, `$derived()`, `$effect()`)
- `*.ts` under `lib/api/` — thin fetch-based API clients, one per backend
- `cn(...)` (`lib/utils.ts`) — `clsx` + `tailwind-merge` helper for composable classes
- shadcn-svelte style: `nova`, base color `neutral`, icons via `@lucide/svelte`
