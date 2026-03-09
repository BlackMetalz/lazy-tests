#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: scripts/check_close_wait.sh <local-port>" >&2
  exit 1
fi

PORT="$1"

if command -v ss >/dev/null 2>&1; then
  count="$(ss -tan state close-wait | awk -v port=":${PORT}" '$4 ~ port"$" || $4 ~ port" " { c++ } END { print c + 0 }')"
  echo "CLOSE_WAIT sockets on local port ${PORT}: ${count}"
  exit 0
fi

if command -v netstat >/dev/null 2>&1; then
  count="$(netstat -tan 2>/dev/null | awk -v port="${PORT}" 'toupper($NF) == "CLOSE_WAIT" && ($4 ~ "\\." port "$" || $4 ~ ":" port "$") { c++ } END { print c + 0 }')"
  echo "CLOSE_WAIT sockets on local port ${PORT}: ${count}"
  exit 0
fi

echo "Neither ss nor netstat is available." >&2
exit 2
