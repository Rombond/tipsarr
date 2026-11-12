# Public demo

`cmd/demo-gateway` gives every visitor a Tipsarr process of their own (admin rights included), started from a seeded
template and deleted when the session ends, so a visitor can only damage their own sandbox.

```
visitor ──► gateway :8080 ──cookie tipsarr_demo──► sandbox (tipsarr, TIPSARR_DEMO_MODE=true, 127.0.0.1:random)
                                                        │ only allowed outbound address
            gateway 127.0.0.1:18200 ◄───────────────────┘
              /jellyfin  fake Jellyfin (real TMDB titles, 4 accounts, watch history, Playback Reporting)
              /radarr /sonarr  read-only fakes
              /tmdb /tmdbimg /boxoffice  caching proxy: holds TMDB_API_KEY, one disk cache, one rate limit
            gateway :9090  /healthz /metrics (private)
```

Demo mode in a sandbox: dry-run forced; TMDB key, Jellyfin key and public URL locked; webhooks offline; teaser and
yt-dlp jobs off; every outbound connection limited to the gateway's internal address; webhook test, push relay,
LDAP, OIDC and resolver calls answer 403 `demo_disabled`; the login page offers "Sign in as admin / as user".

## Sessions: web and apps

- Web: a browser loading a page gets a sandbox and the `tipsarr_demo` cookie. API calls without a session answer
  401 `demo_expired`.
- Apps (Bearer token): `GET /api/v1/status`, `/health`, `/auth/methods` and `/setup/status` without a session are
  answered by any running sandbox (a warm one first) and allocate nothing (503 `demo_starting` while none runs).
  `POST /api/v1/auth/token` (or `/auth/login`) without a session allocates a sandbox, with the same per-IP limit
  (429 `demo_ip_limit`) and cap (503 `demo_full`); a failed sign-in frees it at once. The token returned is mapped to
  the session, so later requests with `Authorization: Bearer <token>` (event stream and images included) reach it.
  When the session ends, the token gets 401 `demo_expired` and the app returns to its sign-in screen.
- Accounts (fake Jellyfin, password `demo` for all): `admin` (admin), `user` (plain user), `carol`, `dave`.
  Push device registration is stored by the sandbox but nothing is ever sent (push cannot be turned on).

## Run locally

```
cd frontend && npm run build && cp -R build/. ../backend/internal/web/dist/ && cd ../backend
go build -o /tmp/tipsarr ./cmd/tipsarr
TMDB_API_KEY=... DEMO_TIPSARR_BIN=/tmp/tipsarr DEMO_DATA_DIR=/tmp/demo-data DEMO_WARM=1 go run ./cmd/demo-gateway
# open http://localhost:8080 ; metrics on http://127.0.0.1:9090/metrics
```

The first start builds the template (`<data>/template`, about one to two minutes); it is rebuilt when the tipsarr
binary, the seed or `DEMO_INTERNAL_ADDR` changes. `demo-gateway seed` only builds it.

## Settings (environment)

| variable | default | |
|---|---|---|
| `TMDB_API_KEY` | (required) | v3 key or v4 token, only used by the gateway |
| `DEMO_LISTEN` / `DEMO_ADMIN_LISTEN` | `:8080` / `127.0.0.1:9090` | public / metrics listeners |
| `DEMO_INTERNAL_ADDR` | `127.0.0.1:18200` | loopback listener of the fakes and the proxy |
| `DEMO_DATA_DIR` | `./demo-data` | template, `cache/`, `sandboxes/` (wiped at start) |
| `DEMO_TIPSARR_BIN` | `tipsarr` | binary started per visitor |
| `DEMO_WARM` / `DEMO_MAX_SESSIONS` | `3` / `20` | started sandboxes waiting / session cap |
| `DEMO_IDLE_TIMEOUT` / `DEMO_MAX_LIFETIME` | `30m` / `60m` | session end |
| `DEMO_SESSIONS_PER_IP` / `DEMO_IP_WINDOW` | `5` / `1h` | new sessions per client address |
| `DEMO_IP_ALLOWLIST` | (none) | IPs, CIDRs and host names (fail2ban `ignoreip` syntax, spaces or commas) exempt from the per-address limit; host names are resolved again every 5 minutes |
| `DEMO_TRUSTED_PROXIES` | (none) | CIDRs whose `X-Forwarded-For` is trusted |
| `DEMO_COOKIE_SECURE` | `false` | always Secure cookie (otherwise only behind https) |
| `DEMO_TMDB_RATE` / `DEMO_TMDB_TTL` | `5` / `24h` | TMDB calls per second for the whole demo / cache lifetime |
| `DEMO_CHILD_MEMORY` | `160MiB` | GOMEMLIMIT of a sandbox (GOMAXPROCS=1) |
| `DEMO_MAX_BODY_BYTES` | `5242880` | request body cap |
| `DEMO_PUBLIC_URL` | | public address shown in links |
