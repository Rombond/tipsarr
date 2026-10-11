# Local test environment

Runs the real Tipsarr server on this machine against fake Jellyfin, TMDB, Radarr, Sonarr and box-office services, so
you can try the web app, the iOS simulator and the Android emulator without touching the NAS.

    dev/run.sh            start (the first run builds, starts and fills it with data)
    dev/run.sh --reset    wipe the data and start fresh
    dev/stop.sh           stop everything

Open the web app at http://localhost:8088. In the iOS simulator use `http://localhost:8088` as the server address; in the
Android emulator use `http://10.0.2.2:8088`.

Accounts (password `test` for all):

| account | what it tests |
|---|---|
| alice | admin: everything, settings, the Users page and permissions |
| bob | default rights, 3 movies per 7 days (2 used), pending, failed and declined requests, watchlist and hidden titles |
| carol | request manager (not an admin): approves requests, sees everyone's, no limit |
| dave | movies only (no shows), 1 movie per day (unused), can report issues |

Things to know:

- Requests you approve "download" on a clock: about 8 seconds waiting, then a movie downloads for about 80 seconds (a
  show season by season, about 30 seconds each), then it is available. The availability alert is held until the library
  shows the title or 10 minutes pass.
- Movie 50 is refused by the fake Radarr, so requesting it ends as a failed request.
- The catalogue is made up: 120 movies, 40 shows, 15 people, with French titles when the language is French.
- The fakes live in `backend/cmd/devenv`; the server finds them through `TIPSARR_TMDB_URL`, `TIPSARR_TMDB_IMAGE_URL`
  and `TIPSARR_BOXOFFICE_URL` (unset = the real services).
- `TIPSARR_FAKE_TRAILER=dQw4w9WgXcQ dev/run.sh` makes every trailer that real YouTube video, to try playback (the default keys are made up and do not play).
- Fixed ports: 8088 Tipsarr, 18096 Jellyfin, 18097 TMDB, 18098 images, 18099 Radarr, 18100 Sonarr, 18101 box office.
