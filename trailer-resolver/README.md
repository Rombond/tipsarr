# trailer-resolver

Optional companion of Tipsarr. It downloads YouTube trailers with [yt-dlp](https://github.com/yt-dlp/yt-dlp) into a folder, so the Teasers tab of the iOS and Android apps can play them in a native player. Without it Tipsarr works as before and the Teasers tab is hidden in the apps.

> **YouTube's terms.** YouTube forbids downloading its content except as the Service allows, and automated access. Running this container is **your decision and your risk**: your address can be rate-limited or blocked. It is off by default, never reachable from outside your network, and the files are only served to signed-in Tipsarr users. The image is published to `ghcr.io/rombond/trailer-resolver` on each `resolver-v*` tag (separate from the Tipsarr release). You can also build it yourself with `docker build -t trailer-resolver:local .`.

## What is inside

Debian slim with the official yt-dlp release binary, a static ffmpeg and ffprobe build from the yt-dlp project, and Deno (the JavaScript runtime yt-dlp needs for YouTube). Each download is checked against the checksums the projects publish. The service itself is a small Go program (`*.go`).

## Build

```bash
docker build -t trailer-resolver:local trailer-resolver
# a fixed yt-dlp version instead of the latest one:
docker build --build-arg YTDLP_VERSION=2026.08.19 -t trailer-resolver:local trailer-resolver
```

amd64 and arm64 are supported (the build picks the right binaries).

## Run (docker compose, next to Tipsarr)

```yaml
services:
  trailer-resolver:
    image: trailer-resolver:local
    container_name: TrailerResolver
    restart: unless-stopped
    environment:
      RESOLVER_API_KEY: ${TRAILER_RESOLVER_KEY}   # a long random string, the same in Tipsarr's settings
      RESOLVER_MAX_HEIGHT: "720"
      RESOLVER_CACHE_GB: "20"
    volumes:
      - ${ROOT_STORAGE}/servarr/tipsarr/trailers:/cache   # videos and their small .json files
  tipsarr:
    environment:
      TIPSARR_TRAILERS_DIR: /trailers
    volumes:
      - ${ROOT_STORAGE}/servarr/tipsarr/trailers:/trailers:ro   # Tipsarr only reads the videos
```

The container runs as user `65532` (it needs to own its tools folder so `yt-dlp -U` can replace the binary), so give that user the shared folder once: `mkdir -p <folder> && chown 65532:65532 <folder>`. The videos are world-readable, so Tipsarr (any user) can read them.

Then in Tipsarr (admin settings, trailer files): turn it on, URL `http://trailer-resolver:8090`, the same key. The apps show the Teasers tab once Tipsarr can reach a healthy resolver.

Do **not** publish port 8090 outside the Docker network.

## Settings (environment)

| Variable | Default | Meaning |
|---|---|---|
| `RESOLVER_API_KEY` | required | Bearer key Tipsarr sends |
| `RESOLVER_ADDR` | `:8090` | Listen address |
| `RESOLVER_CACHE_DIR` | `/cache` | Where videos and `<key>.json` go |
| `RESOLVER_CACHE_GB` | `20` | Size cap; the least recently played videos go first |
| `RESOLVER_UNUSED_DAYS` | `30` | Videos nobody played for this long are deleted |
| `RESOLVER_MAX_HEIGHT` | `720` | Default maximum video height |
| `RESOLVER_HOURLY_CAP` | `60` | Downloads started per hour at most |
| `RESOLVER_PAUSE_MIN_SECONDS` / `_MAX_` | `5` / `15` | Random pause between downloads |
| `RESOLVER_TIMEOUT_SECONDS` | `120` | Per download |
| `RESOLVER_WORKERS` | `1` | Downloads at the same time |
| `RESOLVER_MAX_SECONDS` | `300` | Videos longer than this are refused for good (a trailer is short) |
| `RESOLVER_BREAKER_AFTER` / `_HOURS` | `5` / `6` | After that many temporary failures in a row, pause downloads for that long |
| `RESOLVER_AUTO_UPDATE` | `false` | Run `yt-dlp -U` once a day on its own. Normally off: Tipsarr's job `ytdlp-update` (every 12 hours, only while trailer files are on) asks for the update instead |

## API (internal, `Authorization: Bearer <key>`)

- `POST /v1/resolve` `{"key":"<11 chars>","maxHeight":720}` returns `202` (queued or running) or `200` (ready, or failed for good).
- `GET /v1/status/{key}` returns `{status: queued|running|ready|failed, bytes, height, duration, error, permanent}`.
- `GET /v1/health` returns tool versions, cache size, counts, breaker state and the last error.
- `POST /v1/touch` `{"keys":[...]}` tells the cache which videos were played (for the cleanup order).
- `POST /v1/update` runs `yt-dlp -U` and answers `{before, after, updated}`. `DELETE /v1/cache` empties the cache.
- `GET /healthz` (no key) is only for the Docker health check.

Only an 11-character YouTube key is accepted (never a URL) and yt-dlp is started with an argument list, never a shell.

## Behaviour worth knowing

- Output is H.264 and AAC in MP4, with the index at the start of the file, so playback starts at once.
- Videos that cannot be downloaded for good (private, removed, blocked in your country, age-restricted) are marked failed and not retried. Other errors are retried after 1, 5 and 30 minutes, then given up.
- Repeated temporary failures (typically YouTube blocking the address) open a circuit breaker: no downloads for 6 hours, shown in `/v1/health`.
- State survives a restart (`<key>.json` next to each video).

## Develop

```bash
cd trailer-resolver && go vet ./... && go test -race ./...
```
