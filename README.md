# Tipsarr

Discover, request and track movies and TV shows for your [Jellyfin](https://jellyfin.org) library, in one small self-hosted app. Sign in with your Jellyfin account (or through single sign-on), browse what is popular, see what your server already has, request what it lacks, and let [Radarr](https://radarr.video) and [Sonarr](https://sonarr.tv) do the downloading.

Tipsarr is one Go binary with a Svelte interface built in. It replaces the usual trio of a request manager, a suggestion engine and a box-office tracker.

> **Nothing is ever requested or added automatically.** Every request is a person's action, and **dry-run is on by default**: until you set `TIPSARR_DRY_RUN=false`, Tipsarr never writes anything to Radarr or Sonarr.

## Features

- **One identity: your Jellyfin account.** Password login, or OpenID Connect single sign-on (Authelia, Keycloak…) matched to a Jellyfin user. No separate user database to maintain.
- **Discover and search** through TMDB: trending, popular, upcoming, genres, tags, people, collections, trailers, cast and crew. Search also finds titles by tag ("shark").
- **Availability badges** from your Jellyfin library, kept fresh by sync jobs and an optional Jellyfin webhook.
- **Requests with approval**: users ask, admins approve or decline (with a reason). Admins' own requests are approved at once. Movies go to Radarr, shows to Sonarr, with season selection, quality profile and folder choice, an anime folder, and per-genre folders for movies.
- **Live progress**: states from requested to downloading to available, per-season percentages for shows, pushed to the browser as they happen.
- **Import what Radarr and Sonarr already monitor** so the Requests page shows everything on its way, whoever added it.
- **Netflix-style suggestions per user** built from their watch history ("Because you watched…", popular on your server), never added automatically.
- **Box office**: weekend and full-week charts by region (Box Office Mojo), matched to TMDB, with availability.
- **Profile pictures** from an uploaded image, the `jpegPhoto` of an LDAP/LLDAP account, or Jellyfin, in that order.
- **Watchlist and "not interested"** per user, **issue reporting** (video, audio, subtitles) with comment threads, **user profiles** with statistics.
- **Outgoing webhooks** (ntfy, Discord, anything that takes JSON) signed with HMAC, for requests and issues.
- **SQLite by default**, or PostgreSQL / MySQL.
- **English and French**, built to add more (one file per language).

## Quick start (Docker)

```yaml
# compose.yaml
services:
  tipsarr:
    image: ghcr.io/rombond/tipsarr:latest
    container_name: tipsarr
    restart: unless-stopped
    ports:
      - '8080:8080'
    environment:
      PUID: '911'                                # the user/group that owns ./tipsarr-config (default 65532)
      PGID: '911'
      TIPSARR_DRY_RUN: 'true'                    # keep true until you have tried one real request
      # TIPSARR_JELLYFIN_URL: http://jellyfin:8096
    volumes:
      - ./tipsarr-config:/config
```

```bash
docker compose up -d
```

Open `http://<host>:8080`. The first-run page asks for your Jellyfin address; then sign in with a Jellyfin account. **The first Jellyfin administrator to sign in becomes the Tipsarr admin.**

To build the image yourself: `docker build -t tipsarr .` (or `docker compose -f deployments/docker-compose.dev.yml up --build`).

### First configuration (Settings, as admin)

1. **TMDB**: paste a TMDB API key or read-access token (free at themoviedb.org). Everything metadata-related needs it.
2. **Jellyfin API key** (Jellyfin Dashboard → API Keys): enables library and watch-history sync, user import and single sign-on. Optionally set the public Jellyfin address so the "Play" button opens the right place.
3. **Radarr / Sonarr** (Radarr / Sonarr page): add each instance with its URL and API key, test it, pick the default quality profile and root folder (plus the optional anime folder for Sonarr and genre folders for Radarr). The default instance of each kind receives requests.
4. **Box office regions** (e.g. `US,GB,FR`), then run the `boxoffice-refresh` job once.
5. Optional: **webhooks** for notifications, the **Jellyfin webhook** (below) and **single sign-on**.

Before leaving dry-run: make one request as an admin with `TIPSARR_DRY_RUN=true`, check the logs show what would have been sent, then restart with `TIPSARR_DRY_RUN=false`.

## Configuration

Environment variables (all optional):

| Variable | Default | Meaning |
|---|---|---|
| `TIPSARR_PORT` | `8080` | HTTP port (`TIPSARR_ADDR` sets the full address, e.g. `127.0.0.1:8080`) |
| `TIPSARR_CONFIG_DIR` | `./config` (`/config` in Docker) | SQLite file, image and metadata caches |
| `TIPSARR_DB_URL` | `sqlite:<config>/tipsarr.db` | Or `postgres://user:pass@host:5432/db?sslmode=disable`, or `mysql://user:pass@host:3306/db` |
| `TIPSARR_JELLYFIN_URL` | – | Pre-fills the Jellyfin step of the first-run page |
| `TIPSARR_DRY_RUN` | `true` | `false` lets approved requests reach Radarr/Sonarr |
| `TIPSARR_COOKIE_SECURE` | `false` | Always mark the session cookie `Secure` (otherwise only behind a proxy sending `X-Forwarded-Proto: https`) |
| `TIPSARR_SETUP_TOKEN` | `false` | `true` makes the first-run setup require a one-time token printed in the log; useful if a fresh install is reachable by strangers |
| `TIPSARR_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `PUID` / `PGID` | `65532` | User and group the container runs as, like the `*arr` images. It starts as root, gives `/config` to `PUID:PGID`, then drops to them, so the folder never needs a manual `chown` |

Everything else (API keys, services, regions, single sign-on) is set in the interface and stored in the database. Secrets are write-only: the interface never shows them again.

### Reverse proxy

Put Tipsarr behind your usual reverse proxy for HTTPS. It needs `X-Forwarded-Proto` / `X-Forwarded-Host` (all common proxies send them), and the live-update stream (`/api/v1/events`, Server-Sent Events) must not be buffered: Caddy and Traefik work as is; with nginx add `proxy_buffering off;`.

### Single sign-on (OpenID Connect)

Settings → Single sign-on. Register a client at your provider, then enter its address, client id and secret.

- **Redirect URI:** `https://<tipsarr>/api/v1/auth/oidc/callback` (shown in Settings).
- **Scopes:** `openid profile email groups`; PKCE (S256) is used.
- The provider's `preferred_username` is matched, ignoring case, to a **Jellyfin username**, so those names must agree (they do with a Jellyfin LDAP plugin). A Jellyfin API key must be saved.
- Members of the optional **admin group** (read from the `groups` claim) become admins; Jellyfin admins always are.
- When single sign-on is set up, the password form is hidden. **If it refuses someone** (no matching or a disabled Jellyfin account, provider error, expired attempt), the login page shows why and brings the password form back. `/login?password=1` always shows it, in case the provider is down.

Authelia client example:

```yaml
identity_providers:
  oidc:
    clients:
      - client_id: tipsarr
        client_name: Tipsarr
        client_secret: '$pbkdf2-sha512$…'   # hash of the secret you enter in Tipsarr
        public: false
        authorization_policy: one_factor
        require_pkce: true
        pkce_challenge_method: S256
        redirect_uris: ['https://tipsarr.example.org/api/v1/auth/oidc/callback']
        scopes: [openid, profile, email, groups]
```

Tipsarr itself must be able to reach the provider's address (container to provider), not only your browser.

### Requests: quality profile and folder

Everyone can choose the quality profile when requesting. The folder choice is shown to admins only; Settings → Folder choice lets every user choose it too. Whatever is not chosen uses the Radarr/Sonarr defaults.

### Profile pictures from LDAP (optional)

Settings → Profile pictures from LDAP: the LDAP address (for LLDAP `ldap://host:3890`), a read-only bind account, and the users' base DN (for example `ou=people,dc=example,dc=com`). Tipsarr reads each user's `jpegPhoto` (matched on `uid` = their Jellyfin username), cached for 12 hours. People can also upload their own picture from their profile, which takes priority.

### Jellyfin webhook (optional)

Makes availability and watch history update within seconds instead of at the next hourly sync. Install Jellyfin's Webhook plugin, add a *Generic* destination with the URL and template shown under Settings → Jellyfin webhook (the URL contains a secret token; keep it private).

### Outgoing webhooks

Admin → Webhooks. Events: `request.created|approved|declined|failed`, `media.available`, `issue.created|commented|resolved`. Each delivery is a JSON `POST` with `X-Tipsarr-Event` and, when a secret is set, `X-Tipsarr-Signature: sha256=<hmac of the body>`. A webhook pointing at one of your Radarr/Sonarr instances is refused.

## Security notes

- Sessions are random tokens in an `HttpOnly`, `SameSite=Lax` cookie; only a hash is stored. Failed password logins are rate limited.
- Images are proxied and cached, so browsers only ever talk to Tipsarr.
- Run it on a private network or behind your SSO/reverse proxy. There is no per-user quota system; admins approve what users request.

## Develop

Requirements: Go 1.26+, Node 22+.

```bash
make dev            # backend on :8080 and the Vite dev server on :5173 (proxies /api)
make test           # go test + svelte-check + i18n check
make generate       # re-export the OpenAPI spec and regenerate the frontend TypeScript types
make build          # single Docker image (frontend embedded in the Go binary)
```

- `backend/`: Go with [chi](https://github.com/go-chi/chi), [huma](https://huma.rocks) (code-first OpenAPI) and [bun](https://bun.uptrace.dev). Migrations are portable SQL in `backend/internal/store/migrations`. All writes to Radarr/Sonarr go through one dry-run-aware client.
- `frontend/`: SvelteKit (Svelte 5), Tailwind 4, shadcn-svelte. The API client is generated from the OpenAPI spec. Every string is translated through `t()`; a language is one file in `frontend/src/lib/i18n` plus one line in `LOCALES`, and CI checks that every catalog has the same keys.
- External database tests: set `TIPSARR_TEST_PG_URL` / `TIPSARR_TEST_MYSQL_URL` (CI does this with service containers).

Releases: pushing a `v*` tag publishes `ghcr.io/<owner>/tipsarr` (amd64 and arm64).

## Acknowledgements

Tipsarr is inspired by [Seerr](https://github.com/seerr-team/seerr) (requests), [SuggestArr](https://github.com/giuseppe99barchetta/SuggestArr) (suggestions) and [Boxarr](https://github.com/iongpt/Boxarr) (box office). It is an independent, from-scratch implementation: none of their code is included. Movie and show data comes from [TMDB](https://www.themoviedb.org) (Tipsarr is not endorsed or certified by TMDB); box-office charts from [Box Office Mojo](https://www.boxofficemojo.com).

## License

MIT, see [LICENSE](LICENSE).
