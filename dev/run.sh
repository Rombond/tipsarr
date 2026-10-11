#!/usr/bin/env bash
# Starts the real Tipsarr server on :8088 against fake Jellyfin, TMDB, Radarr, Sonarr and box-office services,
# seeds it with accounts and data, and leaves everything running. `dev/run.sh --reset` starts from a fresh database.
# Stop it with dev/stop.sh. See dev/README.md.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT=$PWD
DEV=$ROOT/dev
PORT=8088
mkdir -p "$DEV/.logs" "$DEV/.bin"

"$DEV/stop.sh" quiet || true

if [ "${1:-}" = "--reset" ]; then rm -rf "$DEV/.data"; fi
FRESH=0
[ -d "$DEV/.data" ] || FRESH=1

# the web app: build it when missing or older than its sources
if [ ! -f frontend/build/index.html ] || [ -n "$(find frontend/src locales -newer frontend/build/index.html -print -quit)" ]; then
  echo "building the web app ..."
  (cd frontend && npm run build >"$DEV/.logs/web-build.log" 2>&1) || { echo "web build failed, see dev/.logs/web-build.log"; exit 1; }
fi
rm -rf backend/internal/web/dist/* 2>/dev/null || true
cp -R frontend/build/. backend/internal/web/dist/

echo "building the server and the fakes ..."
(cd backend && go build -o "$DEV/.bin/tipsarr" ./cmd/tipsarr && go build -o "$DEV/.bin/devenv" ./cmd/devenv)

start() { # name, command...
  local name=$1; shift
  nohup "$@" >"$DEV/.logs/$name.log" 2>&1 &
  echo $! >>"$DEV/.pids"
}
: >"$DEV/.pids"

(cd backend && start devenv "$DEV/.bin/devenv")
for _ in $(seq 1 50); do curl -fs http://127.0.0.1:18096/System/Info/Public >/dev/null 2>&1 && break; sleep 0.2; done

TIPSARR_PORT=$PORT \
TIPSARR_CONFIG_DIR="$DEV/.data" \
TIPSARR_DRY_RUN=false \
TIPSARR_COOKIE_SECURE=false \
TIPSARR_JELLYFIN_URL=http://127.0.0.1:18096 \
TIPSARR_TMDB_URL=http://127.0.0.1:18097 \
TIPSARR_TMDB_IMAGE_URL=http://127.0.0.1:18098 \
TIPSARR_BOXOFFICE_URL=http://127.0.0.1:18101 \
  start tipsarr "$DEV/.bin/tipsarr"
for _ in $(seq 1 100); do curl -fs http://127.0.0.1:$PORT/api/v1/status >/dev/null 2>&1 && break; sleep 0.2; done
curl -fs http://127.0.0.1:$PORT/api/v1/status >/dev/null || { echo "the server did not start, see dev/.logs/tipsarr.log"; exit 1; }

if [ "$FRESH" = 1 ]; then
  echo "seeding ..."
  (cd backend && "$DEV/.bin/devenv" -seed http://127.0.0.1:$PORT) 2>&1 | tee "$DEV/.logs/seed.log" | grep -v '^20[0-9][0-9]/' || true
else
  echo "keeping the data of the last run (dev/run.sh --reset starts fresh)"
  echo "accounts: alice, bob, carol, dave, password test"
fi

cat <<TXT

Tipsarr dev environment is running.
  web                 http://localhost:$PORT
  iOS simulator       http://localhost:$PORT
  Android emulator    http://10.0.2.2:$PORT
logs: dev/.logs/   stop: dev/stop.sh
TXT
