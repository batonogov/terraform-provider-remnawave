#!/usr/bin/env bash
set -euo pipefail

endpoint=${1:-http://localhost:3000}
attempts=${REMNAWAVE_READY_ATTEMPTS:-60}
delay_seconds=${REMNAWAVE_READY_DELAY_SECONDS:-1}

for ((attempt = 1; attempt <= attempts; attempt++)); do
  if curl -fsS \
    -H "X-Forwarded-For: 127.0.0.1" \
    -H "X-Forwarded-Proto: https" \
    -H "X-Remnawave-Client-Type: browser" \
    "${endpoint%/}/api/auth/status" \
    >/dev/null 2>&1; then
    exit 0
  fi
  if ((attempt < attempts)); then
    sleep "$delay_seconds"
  fi
done

echo "Remnawave REST API did not become ready after $attempts attempts" >&2
exit 1
