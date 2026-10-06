# Tipsarr

Self-hosted media discovery and requests for your [Jellyfin](https://jellyfin.org) library: one Go backend, one Svelte frontend, one login (your Jellyfin account). MIT licensed.

> Work in progress (rewrite). Implemented: Jellyfin login, discover / search / details, library and watch-history sync with availability badges, requests with approval (Radarr/Sonarr, anime folder), live updates (SSE), outgoing webhooks, Netflix-style suggestions per user, weekend box office, watchlist and blocklist, users admin. **Dry-run is on by default: nothing is ever sent to Radarr/Sonarr until you set `TIPSARR_DRY_RUN=false`.**

## Run locally

Requirements: Go 1.26+, Node 22+.

```bash
# terminal 1 - backend (http://localhost:8080)
cd backend && go run ./cmd/tipsarr

# terminal 2 - frontend with hot reload (http://localhost:5173, proxies /api to the backend)
cd frontend && npm ci && npm run dev
```

Open http://localhost:5173. On first run the setup page asks for the **setup token** (printed in the backend log at startup, so a stranger cannot claim a fresh install), your Jellyfin URL (and optionally a TMDB API key / read token); then sign in with a Jellyfin account. The first Jellyfin administrator to sign in becomes the Tipsarr admin and can set the TMDB key under Settings.

Backend environment (all optional): `TIPSARR_PORT` (8080), `TIPSARR_CONFIG_DIR` (`./config`), `TIPSARR_DB_URL` (default `sqlite:<config>/tipsarr.db`; also `postgres://…` or `mysql://user:pass@host:3306/db`), `TIPSARR_JELLYFIN_URL`, `TIPSARR_DRY_RUN` (default `true`), `TIPSARR_COOKIE_SECURE` (`true` to always mark the session cookie Secure; otherwise it is Secure behind a proxy that sends `X-Forwarded-Proto: https`), `TIPSARR_LOG_LEVEL`.

## Docker (single image)

```bash
docker compose -f deployments/docker-compose.dev.yml up --build     # http://localhost:8080
```

The image contains the Go binary with the built frontend embedded; data lives in the `/config` volume.

## Develop

```bash
make test       # go test + svelte-check
make generate   # re-export the OpenAPI spec and regenerate the frontend TypeScript types
make build      # single Docker image (frontend embedded in the Go binary)
```

## License

MIT, see [LICENSE](LICENSE).
