<p align="center">
  <img src="frontend/static/icon.svg" alt="Tipsarr logo" width="112" height="112">
</p>

<h1 align="center">Tipsarr</h1>

<p align="center">
  Discover, request and follow movies and TV shows for your <a href="https://jellyfin.org">Jellyfin</a> server.<br>
  One self-hosted app with a web interface and native iOS and Android apps.
</p>

<p align="center">
  <a href="https://tipsarr-demo.brebond.com"><img alt="Try the demo" src="https://img.shields.io/badge/demo-try%20it%20now-4f46e5?style=for-the-badge"></a>
  <a href="https://github.com/Rombond/tipsarr/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/Rombond/tipsarr/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/Rombond/tipsarr/releases"><img alt="Latest release" src="https://img.shields.io/github/v/release/Rombond/tipsarr"></a>
  <a href="LICENSE"><img alt="MIT license" src="https://img.shields.io/github/license/Rombond/tipsarr"></a>
</p>

<!-- TODO: screenshots. Add docs/images/web-discover.png, web-request.png, ios-*.png, android-*.png and replace this comment
     with an image row (no real account, library or server address visible). -->

**[Try the live demo](https://tipsarr-demo.brebond.com)**: every visitor gets a private, throwaway Tipsarr with sample
data and admin access. Nothing you do there is seen by anyone else; the sandbox is deleted after 30 minutes without
activity (60 minutes at most).

Tipsarr is a single Go binary with the web interface built in. Sign in with your Jellyfin account, browse what is
popular, see what your server already has, ask for what it lacks, and let [Radarr](https://radarr.video) and
[Sonarr](https://sonarr.tv) do the downloading once an admin approves.

> [!IMPORTANT]
> **Nothing is ever requested or added automatically**, and **dry-run is on by default**: until you set
> `TIPSARR_DRY_RUN=false`, Tipsarr never writes anything to Radarr or Sonarr. Try a request first, read the log, then
> turn it off.

## Features

- **Discovery**: trending, popular, upcoming, genres, keywords, people, collections, trailers, cast and crew, all from
  TMDB. Search also matches keywords ("shark").
- **Personal suggestions** built from each person's Jellyfin watch history ("Because you watched…", popular on this
  server). Suggestions are only suggestions: nothing is requested for you.
- **Box office**: weekend and weekly charts per region (Box Office Mojo), matched to TMDB, with availability.
- **Requests** to Radarr (movies) and Sonarr (shows, with season selection): approval by an admin or a request manager,
  decline with a reason, quality profile and folder choice, anime folder, per-genre movie folders, **4K requests** (a
  request is 4K when its quality profile allows 2160p). Progress (requested, downloading per season, available) is
  pushed live to the browser and the apps.
- **Import** of what Radarr and Sonarr already monitor, so the Requests page shows everything that is on its way.
- **Jellyfin integration**: login with Jellyfin accounts, availability badges from your library, watch history sync,
  optional Jellyfin webhook for instant updates, user import, "Play" links to Jellyfin.
- **Statistics**: time watched, genres, decades, most watched titles, "asked for, never watched"; exact numbers when
  Jellyfin's Playback Reporting plugin is installed, estimates otherwise.
- **Teasers** *(experimental, mobile apps)*: a vertical feed of trailers. Needs the optional
  [`trailer-resolver`](trailer-resolver/README.md) sidecar; off until an admin enables it.
- **Issues**: report a video, audio or subtitle problem on a title, with comment threads and resolution.
- **Watchlist** and "not interested" per person, profiles with statistics, profile pictures (upload, LDAP `jpegPhoto`
  or Jellyfin).
- **Permissions and quotas**: Seerr-compatible permission flags (request, request movies/shows only, 4K, auto-approve,
  manage requests, issues, users, settings) and per-person request limits (N movies or seasons per X days).
- **Single sign-on** with OpenID Connect (Authelia, Authentik, Keycloak…), mapped to Jellyfin accounts. LDAP users
  work through Jellyfin's LDAP plugin; Tipsarr can read their profile pictures from LDAP.
- **Notifications**: outgoing webhooks with Liquid templates and presets for Discord, ntfy, Gotify, Telegram, Slack,
  Pushover, Apprise and Home Assistant, HMAC signatures, per-recipient delivery.
- **Native iOS and Android apps** with push notifications through a privacy-preserving relay (see below).
- **Read-only API key** for dashboards (Homepage, Glance…), full OpenAPI spec.
- **English and French**; a new language is a folder of JSON files ([how to translate](CONTRIBUTING-TRANSLATIONS.md)).
- **SQLite by default**, or PostgreSQL / MySQL. amd64 and arm64 images.

## What it replaces

Tipsarr covers in one app what is often run as three. It was inspired by these projects; it is an independent
implementation, not a fork.

| | Tipsarr | [Seerr](https://github.com/seerr-team/seerr) (formerly Jellyseerr/Overseerr) | [SuggestArr](https://github.com/giuseppe99barchetta/SuggestArr) | [Boxarr](https://github.com/iongpt/Boxarr) |
|---|---|---|---|---|
| Discover and request | yes | yes | – | – |
| Suggestions from watch history | yes, shown to the person, never requested | – | yes, requests them automatically | – |
| Box-office charts | yes, shown, never added | – | – | yes, can add to Radarr |
| Media servers | Jellyfin only | Jellyfin, Plex, Emby | Jellyfin, Plex, Emby | – (Radarr) |
| Radarr/Sonarr instances | one default per kind, 4K by quality profile | several, separate 4K instances | – | Radarr |
| Native mobile apps | iOS and Android | web app (PWA) | – | – |
| Notifications | webhooks (with presets) and mobile push | many built-in agents | – | – |

If you need Plex or Emby, or separate 4K Radarr/Sonarr instances, Seerr is the better choice today.

## Quick start (Docker Compose)

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
      PUID: '1000'              # owner of ./config (default 65532)
      PGID: '1000'
      TIPSARR_DRY_RUN: 'true'   # keep true until you have tried one request
      # TIPSARR_JELLYFIN_URL: http://jellyfin:8096
    volumes:
      - ./config:/config
```

```bash
docker compose up -d
```

Open `http://<host>:8080`, enter your Jellyfin address, and sign in with a Jellyfin account. **The first Jellyfin
administrator who signs in becomes the Tipsarr admin.** Then, in Settings: paste a TMDB API key, a Jellyfin API key, add
Radarr and Sonarr. The full walkthrough is in [docs/installation.md](docs/installation.md).

## Mobile apps and push notifications

The iOS and Android apps connect to your own Tipsarr server (several servers or accounts are supported) and use the
same API as the web interface.

<!-- TODO: store links once published. -->
- iOS (App Store): *coming soon*
- Android (Google Play): *coming soon*

Apple and Google only deliver notifications signed with the app publisher's credentials, so your server cannot push to
a store app directly. Tipsarr goes through a small relay run by the publisher,
[push-relay](https://github.com/Rombond/push-relay), at `https://push.brebond.com`:

```
your Tipsarr server ──(event + id)──► push.brebond.com ──► APNs / FCM ──► phone ──► your server (details)
```

- The relay receives the device token, the event name (`request.approved`…), an id, the language and the server's host
  name, and builds a **generic text** from its own templates ("Your request was approved"). Titles, user names and
  posters never go through it; the app fetches the details from your server when you open the notification.
- Device tokens are not stored by the relay. Admins turn push on with one click (Settings → Push notifications), which
  gets the server its own key; a key from the public relay has a daily quota of alerts, shown in Settings.
- Prefer not to use it? Leave push off and use [webhooks](docs/notifications.md) (ntfy, Gotify, Pushover…).

Details: [docs/push-notifications.md](docs/push-notifications.md) and [PRIVACY.md](PRIVACY.md).

## Documentation

- [Installation](docs/installation.md): Compose, environment variables, reverse proxy, Jellyfin, Radarr and Sonarr
- [Configuration and admin settings](docs/configuration.md)
- [Authentication](docs/authentication.md): Jellyfin, OpenID Connect, LDAP
- [Notifications (webhooks)](docs/notifications.md) and [push notifications](docs/push-notifications.md)
- [Mobile apps](docs/mobile-apps.md)
- [API](docs/api.md)
- [Upgrading](docs/upgrading.md)
- [FAQ and troubleshooting](docs/faq.md)
- [The demo](docs/demo.md), and how to run your own
- All pages: [docs/](docs/README.md)

## Contributing

Bug reports, ideas, translations and pull requests are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) (development
setup, tests, commit style) and [CONTRIBUTING-TRANSLATIONS.md](CONTRIBUTING-TRANSLATIONS.md). Security problems:
see [SECURITY.md](SECURITY.md), please do not open a public issue.

## Acknowledgements

- [Seerr](https://github.com/seerr-team/seerr) (MIT) for the request workflow, the permission model and Liquid
  webhooks; [SuggestArr](https://github.com/giuseppe99barchetta/SuggestArr) (MIT) for suggestions from watch history;
  [Boxarr](https://github.com/iongpt/Boxarr) for the box-office idea. No Boxarr code is included.
- Box-office figures come from [Box Office Mojo](https://www.boxofficemojo.com).

Movie and show data comes from [TMDB](https://www.themoviedb.org). **This product uses the TMDB API but is not endorsed or certified by TMDB.**

## License

[MIT](LICENSE).
