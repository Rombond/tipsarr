# Configuration and admin settings

Everything below is done in the interface by an admin. Secrets (API keys, passwords) are write-only: once saved they
are never shown again.

## First configuration

1. **Settings → API keys**: a TMDB API key (v3) or read-access token (v4), and a Jellyfin API key.
2. **Settings → Jellyfin links**: the public Jellyfin address for "Play" buttons.
3. **Radarr / Sonarr**: add the instances, see [installation](installation.md#radarr-and-sonarr).
4. **Box office**: choose the regions (e.g. `US,GB,FR`), then run the `boxoffice-refresh` job once (Jobs page).
5. Optional: [webhooks](notifications.md), [push notifications](push-notifications.md),
   [single sign-on](authentication.md), Jellyfin webhook, LDAP profile pictures, read-only [API key](api.md).

## Users

People appear when they first sign in, or all at once with Users → Import from Jellyfin. A Jellyfin administrator is
always a Tipsarr admin. A user disabled or deleted in Jellyfin can no longer sign in.

## Permissions

Each person has a set of rights (the bit values are the same as Seerr's, so a mask can be copied across):

| Group | Rights |
|---|---|
| Admin tools | manage settings, manage users |
| Requests | request (movies, shows), request 4K (movies, shows), choose the folder, see everyone's requests, manage requests (approve, decline, edit, no limit) |
| Auto-approve | auto-approve (movies, shows), auto-approve 4K (movies, shows) |
| Issues | report issues, see everyone's issues, manage issues |

Admins can do everything. The rights and request limits of people who sign in for the first time are set under
Users → Defaults for new people.

## Request limits

Users → a person → Request limits: a number of movies and of TV show seasons per number of days. Empty = the default,
`0` = no limit. People with "manage requests" have no limit.

## Recommendations from other accounts

Users → Recommendations: a person's suggestions can also use the watch history of other accounts (for example a shared
household account). Their watched titles are hidden; only the person's own "not interested" list applies.

## Requests: quality and folder

Everyone can choose the quality profile when requesting. The folder choice is shown to people with the "choose the
folder" right (and admins); Settings → Folder choice can give it to everyone. Whatever is not chosen uses the
Radarr/Sonarr defaults.

Radarr/Sonarr instances: the connection test reaches LAN addresses but refuses loopback (`127.0.0.1`, `localhost`)
and link-local addresses; use the container name or LAN address. Changing the host of an instance asks for the API
key again (the saved key is only ever sent to the host it was saved for).

## Import from Radarr and Sonarr

Settings → Import: titles Radarr and Sonarr already monitor but have not downloaded appear in Requests as approved, so
everyone can follow them. The `servarr-import` job runs every hour.

## Profile pictures from LDAP

Settings → Profile pictures from LDAP: address (`ldap://host:3890` for LLDAP), a read-only bind account and the users'
base DN. Tipsarr reads `jpegPhoto` where `uid` equals the Jellyfin username (cached 12 hours). "Import from Jellyfin"
copies the settings from Jellyfin's LDAP plugin. An uploaded picture takes priority. Pictures larger than 4096 × 4096 pixels (16 MP) or 5 MB are refused.

## Jobs

| Job | Every | |
|---|---|---|
| `library-sync` | 6 h | Jellyfin library (availability) |
| `history-sync` | 1 h | watch history (suggestions, stats) |
| `playback-sync` | 1 h | Playback Reporting data, if the plugin is installed |
| `request-poll` | 15 s | progress of requests in Radarr/Sonarr |
| `servarr-import` | 1 h | import from Radarr/Sonarr |
| `boxoffice-refresh` | 12 h | box-office charts |
| `teaser-pool`, `trailer-files-sync`, `ytdlp-update` | | Teasers, only when enabled |
| `housekeeping` | 1 h | expired sessions and cache |

## Dry-run

The Settings status card shows whether dry-run is on. Dry-run is set by the `TIPSARR_DRY_RUN` environment variable
only, not in the interface, so a click cannot turn it off by accident.
