#!/usr/bin/env bash
# Stops what dev/run.sh started.
cd "$(dirname "$0")"
if [ -f .pids ]; then
  while read -r pid; do kill "$pid" 2>/dev/null || true; done <.pids
  rm -f .pids
  [ "${1:-}" = quiet ] || echo "stopped"
else
  [ "${1:-}" = quiet ] || echo "nothing running"
fi
