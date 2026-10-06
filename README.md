# Tipsarr

Self-hosted media discovery and requests for your [Jellyfin](https://jellyfin.org) library: one Go backend, one Svelte frontend, one login (your Jellyfin account). MIT licensed.

> Work in progress (rewrite). Currently implemented: Jellyfin login, first-run setup, discover / search / details / person / collection pages (TMDB), image cache. Requests, suggestions and box office come next. **Dry-run is on by default: nothing is ever sent to Radarr/Sonarr.**

## Run locally

Requirements: Go 1.26+, Node 22+.

```bash
# terminal 1 - backend (http://localhost:8080)
cd backend && go run ./cmd/tipsarr

# terminal 2 - frontend with hot reload (http://localhost:5173, proxies /api to the backend)
cd frontend && npm ci && npm run dev
```

Open http://localhost:5173. On first run the setup page asks for your Jellyfin URL (and optionally a TMDB API key / read token); then sign in with a Jellyfin account. The first Jellyfin administrator to sign in becomes the Tipsarr admin and can set the TMDB key under Settings.

Backend environment (all optional): `TIPSARR_PORT` (8080), `TIPSARR_CONFIG_DIR` (`./config`), `TIPSARR_DB_URL` (default `sqlite:<config>/tipsarr.db`; also `postgres://…` or `mysql://user:pass@host:3306/db`), `TIPSARR_JELLYFIN_URL`, `TIPSARR_DRY_RUN` (default `true`), `TIPSARR_LOG_LEVEL`.

## Develop

```bash
make test       # go test + svelte-check
make generate   # re-export the OpenAPI spec and regenerate the frontend TypeScript types
make build      # single Docker image (frontend embedded in the Go binary)
```

## License

MIT, see [LICENSE](LICENSE).
