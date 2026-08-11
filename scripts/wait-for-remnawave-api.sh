#!/usr/bin/env bash
set -euo pipefail

endpoint=${1:-http://localhost:3000}
attempts=${REMNAWAVE_READY_ATTEMPTS:-60}
delay_seconds=${REMNAWAVE_READY_DELAY_SECONDS:-1}
timeout_seconds=${REMNAWAVE_READY_TIMEOUT_SECONDS:-90}

validate_integer() {
  local name=$1 value=$2 minimum=$3 maximum=$4
  if [[ ! $value =~ ^[0-9]+$ ]] || ((value < minimum || value > maximum)); then
    printf '%s must be an integer between %d and %d\n' "$name" "$minimum" "$maximum" >&2
    exit 2
  fi
}

validate_integer REMNAWAVE_READY_ATTEMPTS "$attempts" 1 300
validate_integer REMNAWAVE_READY_DELAY_SECONDS "$delay_seconds" 0 10
validate_integer REMNAWAVE_READY_TIMEOUT_SECONDS "$timeout_seconds" 1 300

case "$endpoint" in
  http://* | https://*) ;;
  *)
    echo "Remnawave endpoint must use http:// or https://" >&2
    exit 2
    ;;
esac

SECONDS=0
for ((attempt = 1; attempt <= attempts; attempt++)); do
  remaining_seconds=$((timeout_seconds - SECONDS))
  if ((remaining_seconds <= 0)); then
    break
  fi
  curl_timeout=$remaining_seconds
  if ((curl_timeout > 2)); then
    curl_timeout=2
  fi
  connect_timeout=$curl_timeout
  if ((connect_timeout > 1)); then
    connect_timeout=1
  fi

  if curl -q --fail --silent --show-error \
    --connect-timeout "$connect_timeout" \
    --max-time "$curl_timeout" \
    -H "X-Forwarded-For: 127.0.0.1" \
    -H "X-Forwarded-Proto: https" \
    -H "X-Remnawave-Client-Type: browser" \
    -- "${endpoint%/}/api/auth/status" \
    >/dev/null 2>&1; then
    exit 0
  fi

  remaining_seconds=$((timeout_seconds - SECONDS))
  if ((attempt >= attempts || remaining_seconds <= 0)); then
    break
  fi
  sleep_seconds=$delay_seconds
  if ((sleep_seconds > remaining_seconds)); then
    sleep_seconds=$remaining_seconds
  fi
  if ((sleep_seconds > 0)); then
    sleep "$sleep_seconds"
  fi
done

printf 'Remnawave REST API did not become ready within %d seconds (%d attempts max)\n' \
  "$timeout_seconds" "$attempts" >&2
exit 1
