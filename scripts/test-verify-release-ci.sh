#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-gate-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

tested_sha=1111111111111111111111111111111111111111

jq -n --argjson versions "$(jq '[.versions[] | select(.supported) | .version | ltrimstr("v")]' "$repository_dir/compat-versions.json")" '
  {
    jobs: (
      [
        "Lint",
        "Build",
        "Unit Tests",
        "Documentation",
        "Prepare Compatibility Matrix",
        "Release Gate Tests",
        "Release Artifact Tests",
        "Repository Policy Tests",
        "Vulnerability Scan"
      ] + [$versions[] | "Acceptance Tests (\(.))"]
      | map({name: ., status: "completed", conclusion: "success"})
    )
  }
' >"$temporary_dir/success.json"

run_gate() {
  (
    cd "$repository_dir"
    CI_JOBS_FILE="${CI_JOBS_FILE:-$temporary_dir/success.json}" \
      CI_RUN_CONCLUSION="${CI_RUN_CONCLUSION:-success}" \
      CI_RUN_EVENT="${CI_RUN_EVENT:-push}" \
      CI_RUN_HEAD_BRANCH="${CI_RUN_HEAD_BRANCH:-main}" \
      CI_RUN_HEAD_SHA="${CI_RUN_HEAD_SHA:-$tested_sha}" \
      CHECKED_OUT_SHA="${CHECKED_OUT_SHA:-$tested_sha}" \
      REMOTE_MAIN_SHA="${REMOTE_MAIN_SHA:-$tested_sha}" \
      "$script_dir/verify-release-ci.sh"
  )
}

expect_failure() {
  local name=$1
  shift
  if (export "$@"; run_gate) >/dev/null 2>&1; then
    echo "$name: gate unexpectedly succeeded" >&2
    exit 1
  fi
}

run_gate >/dev/null

jq '(.jobs[] | select(.name == "Unit Tests")).conclusion = "failure"' \
  "$temporary_dir/success.json" >"$temporary_dir/failed.json"
expect_failure "failed job" CI_JOBS_FILE="$temporary_dir/failed.json"

jq '(.jobs[] | select(.name == "Documentation")) |=
  (.status = "completed" | .conclusion = "skipped")' \
  "$temporary_dir/success.json" >"$temporary_dir/skipped.json"
expect_failure "skipped job" CI_JOBS_FILE="$temporary_dir/skipped.json"

jq 'del(.jobs[] | select(.name == "Lint"))' \
  "$temporary_dir/success.json" >"$temporary_dir/missing.json"
expect_failure "missing job" CI_JOBS_FILE="$temporary_dir/missing.json"

expect_failure "cancelled workflow" CI_RUN_CONCLUSION=cancelled
expect_failure "wrong event" CI_RUN_EVENT=pull_request
expect_failure "mismatched checkout" CHECKED_OUT_SHA=2222222222222222222222222222222222222222
expect_failure "stale main" REMOTE_MAIN_SHA=3333333333333333333333333333333333333333

echo "release gate tests passed"
