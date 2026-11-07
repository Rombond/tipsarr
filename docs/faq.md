# FAQ and troubleshooting

**I approved a request and nothing appears in Radarr/Sonarr.**
Dry-run is on by default. Settings shows "Dry-run on"; the log says what would have been sent. Set
`TIPSARR_DRY_RUN=false` and restart the container.

**Everything is "not available" although the title is in Jellyfin.**
Save a Jellyfin API key, then run `library-sync` on the Jobs page. Add the Jellyfin webhook for instant updates
([installation](installation.md#jellyfin)).

**No posters, no search results.**
The TMDB key is missing or wrong (Settings → API keys). Tipsarr must reach `api.themoviedb.org` and
`image.tmdb.org`.

**Request progress or the live updates freeze behind nginx.**
Add `proxy_buffering off;` for `/api/v1/events` (Server-Sent Events).

**Login loops or the session is lost behind a reverse proxy.**
The proxy must send `X-Forwarded-Proto` and `X-Forwarded-Host`. If Tipsarr is served only over HTTPS, set
`TIPSARR_COOKIE_SECURE=true`.

**Single sign-on says no account matches.**
The provider's `preferred_username` (or the configured username claim) must equal a Jellyfin username exactly, case
included, and a Jellyfin API key must be
saved. Use `/login?password=1` to sign in with a password meanwhile. See [authentication](authentication.md).

**Single sign-on fails with a connection error.**
The Tipsarr container itself must reach the provider's address (Docker networking, internal DNS, TLS certificate).

**The "Play" button opens the wrong address.**
Set the public Jellyfin address in Settings → Jellyfin links.

**Box office is empty.**
Choose the regions in Settings, then run `boxoffice-refresh` once.

**Stats show "estimated".**
Install Jellyfin's Playback Reporting plugin for exact watch time.

**Push notifications stopped.**
Settings → Push notifications shows if the daily quota is reached (push resumes at UTC midnight). Check that the phone
allows notifications for the app and that the category is on in the app. Nothing is sent while dry-run is on.

**Permission errors on `/config` at start.**
Set `PUID`/`PGID` to the owner you want; the container fixes the ownership of `/config` itself when it starts as root.
If you run it with `user:`, the folder must already belong to that user.

**Where are the logs?** `docker logs tipsarr`. `TIPSARR_LOG_LEVEL=debug` for more.

**Which Jellyfin versions?** <!-- TODO: state the minimum Jellyfin version tested. -->

Still stuck? Search the [issues](https://github.com/Rombond/tipsarr/issues) or open one with the bug report form.
