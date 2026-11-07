# Installation

Tipsarr ships as one Docker image, `ghcr.io/rombond/tipsarr:latest` (linux/amd64 and linux/arm64). It needs:

- a [Jellyfin](https://jellyfin.org) server it can reach over HTTP;
- a free [TMDB](https://www.themoviedb.org/settings/api) API key or read-access token;
- optionally [Radarr](https://radarr.video) and [Sonarr](https://sonarr.tv) (without them, Tipsarr is a discovery app).

## Docker Compose

```yaml
services:
  tipsarr:
    image: ghcr.io/rombond/tipsarr:latest
    container_name: tipsarr
    restart: unless-stopped
    ports:
      - '8080:8080'
    environment:
      PUID: '1000'
      PGID: '1000'
      TIPSARR_DRY_RUN: 'true'
      # TIPSARR_JELLYFIN_URL: http://jellyfin:8096
      # TIPSARR_DB_URL: postgres://tipsarr:secret@postgres:5432/tipsarr?sslmode=disable
    volumes:
      - ./config:/config
```

`docker compose up -d`, then open `http://<host>:8080`. The first-run page asks for the Jellyfin address (pre-filled
from `TIPSARR_JELLYFIN_URL`); sign in with a Jellyfin account. **The first Jellyfin administrator who signs in becomes
the Tipsarr admin.** Continue with [Configuration](configuration.md).

The container starts as root only to give `/config` to `PUID:PGID` (default `65532`), then drops to that user, like the
`*arr` images. You never need to `chown` the folder.

A ready-made file is in [`deployments/docker-compose.yml`](../deployments/docker-compose.yml). To build the image
yourself: `docker build -t tipsarr .` from the repository root.

### Without Docker

Build the frontend and embed it in the binary (Go 1.26+, Node 22+):

```bash
cd frontend && npm ci && npm run build && cp -R build/. ../backend/internal/web/dist/
cd ../backend && CGO_ENABLED=0 go build -o tipsarr ./cmd/tipsarr
TIPSARR_CONFIG_DIR=/var/lib/tipsarr ./tipsarr
```

## Environment variables

All optional. Everything else (API keys, Radarr/Sonarr, single sign-on, webhooks, push) is set in the interface and
stored in the database.

| Variable | Default | Meaning |
|---|---|---|
| `TIPSARR_PORT` | `8080` | HTTP port |
| `TIPSARR_ADDR` | `:<TIPSARR_PORT>` | Full listen address, e.g. `127.0.0.1:8080` (overrides `TIPSARR_PORT`) |
| `TIPSARR_CONFIG_DIR` | `./config` (`/config` in the image) | SQLite database, image and metadata caches |
| `TIPSARR_DB_URL` | `sqlite:<config>/tipsarr.db` | Or `postgres://user:pass@host:5432/db?sslmode=disable`, or `mysql://user:pass@host:3306/db` |
| `TIPSARR_JELLYFIN_URL` | – | Pre-fills the Jellyfin step of the first-run page |
| `TIPSARR_DRY_RUN` | `true` | `false` (or `0`) lets approved requests reach Radarr/Sonarr and push notifications leave |
| `TIPSARR_COOKIE_SECURE` | unset | `true` always marks the session cookie `Secure`; otherwise it is `Secure` over HTTPS or behind a proxy sending `X-Forwarded-Proto: https` |
| `TIPSARR_SETUP_TOKEN` | `true` (`false` when `TIPSARR_JELLYFIN_URL` is set) | The first-run page asks for a one-time token printed in the log. `false` turns it off |
| `TIPSARR_TRUSTED_PROXIES` | `10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,127.0.0.0/8,::1/128,fc00::/7` | Proxies (CIDRs or addresses) whose `X-Forwarded-For` / `X-Real-IP` is believed; set but empty = none. See [authentication](authentication.md#client-address) |
| `TIPSARR_SESSION_TTL` | `10d` | How long a session lasts after its last use (`36h`, `7d`...) |
| `TIPSARR_SESSION_RECHECK_GRACE` | `24h` | How long sessions keep working while Jellyfin is unreachable; then they are refused until it answers |
| `TIPSARR_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `TIPSARR_TRAILERS_DIR` | `<config>/trailers` | Folder shared with the optional `trailer-resolver` sidecar (Teasers), mounted read-only |
| `PUID` / `PGID` | `65532` | User and group the container runs as |

Variables used only for development and the demo (`TIPSARR_TMDB_URL`, `TIPSARR_TMDB_IMAGE_URL`, `TIPSARR_BOXOFFICE_URL`,
`TIPSARR_FAKE_TRAILER`, `TIPSARR_DEMO_*`, `TIPSARR_TEST_*`) are described in [`dev/README.md`](../dev/README.md) and
[demo.md](demo.md); leave them unset in production.

## Reverse proxy

Tipsarr reads `X-Forwarded-Proto` and `X-Forwarded-Host`, which every common proxy sends. The live-update stream
(`/api/v1/events`, Server-Sent Events) must not be buffered.

**Caddy**

```caddyfile
tipsarr.example.org {
	reverse_proxy tipsarr:8080
}
```

**nginx**

```nginx
server {
    listen 443 ssl;
    server_name tipsarr.example.org;
    # ssl_certificate ...; ssl_certificate_key ...;
    client_max_body_size 10m;          # profile picture uploads

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_buffering off;           # Server-Sent Events
        proxy_read_timeout 1h;
    }
}
```

**Traefik** (labels)

```yaml
    labels:
      - traefik.enable=true
      - traefik.http.routers.tipsarr.rule=Host(`tipsarr.example.org`)
      - traefik.http.routers.tipsarr.entrypoints=websecure
      - traefik.http.routers.tipsarr.tls.certresolver=letsencrypt
      - traefik.http.services.tipsarr.loadbalancer.server.port=8080
```

If you put an authentication proxy (Authelia, Authentik forward auth) in front of Tipsarr, the mobile apps will not be
able to sign in: let `/api/` through and use Tipsarr's own [OpenID Connect](authentication.md) support instead.
<!-- TODO: confirm the exact paths the apps need if a forward-auth bypass is set up. -->

## Jellyfin

1. **API key**: Jellyfin Dashboard → API Keys → add one, paste it in Tipsarr Settings. It enables library and watch
   history sync, user import and single sign-on.
2. **Public address** (Settings): the address people use to open Jellyfin, for the "Play" buttons.
3. **Webhook (optional)**: makes availability and watch history update within seconds instead of at the next sync.
   Install the *Webhook* plugin from Jellyfin's plugin catalogue, add a *Generic* destination with the URL and template
   shown under Settings → Jellyfin webhook, and add the request header `X-Tipsarr-Token` with the secret shown there;
   keep it private. The older form with the secret in the address (`?token=…`) still works but is deprecated (it ends
   up in proxy logs; Tipsarr logs a warning once) and will be removed in a future major version.
4. **Playback Reporting (optional)**: with Jellyfin's Playback Reporting plugin installed, the Stats page shows exact
   watch time instead of estimates.

Without a webhook, the library is synced every 6 hours and the watch history every hour (Jobs page: run them now).

## Radarr and Sonarr

Admin → Radarr / Sonarr: add each instance with its URL and API key (Radarr/Sonarr → Settings → General), press Test,
then choose the default quality profile and root folder. Sonarr can have an anime folder, Radarr per-genre folders. The
default instance of each kind receives the requests. An optional public URL per instance enables "Open in Radarr/Sonarr"
buttons for admins.

4K: there is no separate 4K instance. A request is 4K when its quality profile allows a 2160p quality; who may pick such
a profile is a [permission](configuration.md#permissions).

Before leaving dry-run: make one request as admin with `TIPSARR_DRY_RUN=true`, check the log shows what would have been
sent, then restart with `TIPSARR_DRY_RUN=false`.

## Teasers (optional, experimental)

The Teasers tab of the mobile apps plays trailer files downloaded by the separate
[`trailer-resolver`](../trailer-resolver/README.md) container (yt-dlp). Read its README first: downloading from YouTube
is against YouTube's terms and is your decision. Without it the tab is hidden.
