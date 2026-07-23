#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "release gate: $*" >&2
  exit 1
}

: "${CI_JOBS_FILE:?CI_JOBS_FILE is required}"
: "${CI_RUN_CONCLUSION:?CI_RUN_CONCLUSION is required}"
: "${CI_RUN_EVENT:?CI_RUN_EVENT is required}"
: "${CI_RUN_HEAD_BRANCH:?CI_RUN_HEAD_BRANCH is required}"
: "${CI_RUN_HEAD_SHA:?CI_RUN_HEAD_SHA is required}"
: "${CHECKED_OUT_SHA:?CHECKED_OUT_SHA is required}"
: "${REMOTE_MAIN_SHA:?REMOTE_MAIN_SHA is required}"

[[ "$CI_RUN_CONCLUSION" == "success" ]] ||
  fail "CI workflow conclusion is $CI_RUN_CONCLUSION, want success"
[[ "$CI_RUN_EVENT" == "push" ]] ||
  fail "CI workflow event is $CI_RUN_EVENT, want push"
[[ "$CI_RUN_HEAD_BRANCH" == "main" ]] ||
  fail "CI workflow branch is $CI_RUN_HEAD_BRANCH, want main"
[[ "$CI_RUN_HEAD_SHA" =~ ^[0-9a-f]{40}$ ]] ||
  fail "CI workflow head SHA is not a full commit SHA"
[[ "$CHECKED_OUT_SHA" == "$CI_RUN_HEAD_SHA" ]] ||
  fail "checked-out SHA does not match the CI-tested SHA"
[[ "$REMOTE_MAIN_SHA" == "$CI_RUN_HEAD_SHA" ]] ||
  fail "CI-tested SHA is stale relative to origin/main"

jq -e '.jobs | type == "array" and length > 0' "$CI_JOBS_FILE" >/dev/null ||
  fail "CI jobs response is missing or empty"

non_successful_jobs=$(jq -r '
  .jobs[]
  | select(.status != "completed" or .conclusion != "success")
  | "\(.name): status=\(.status), conclusion=\(.conclusion)"
' "$CI_JOBS_FILE")
[[ -z "$non_successful_jobs" ]] ||
  fail "CI contains non-successful jobs: ${non_successful_jobs//$'\n'/; }"

required_jobs=(
  "Lint"
  "Build"
  "Unit Tests"
  "Documentation"
  "Prepare Compatibility Matrix"
  "Release Gate Tests"
)

while IFS= read -r label; do
  required_jobs+=("Acceptance Tests ($label)")
done < <(jq -r '.versions[] | select(.supported) | .version | ltrimstr("v")' compat-versions.json)

for required_job in "${required_jobs[@]}"; do
  matches=$(jq --arg name "$required_job" '
    [.jobs[] | select(.name == $name and .status == "completed" and .conclusion == "success")]
    | length
  ' "$CI_JOBS_FILE")
  [[ "$matches" == "1" ]] ||
    fail "required job $required_job has $matches successful results, want exactly 1"
done

echo "release gate: CI run validated for $CI_RUN_HEAD_SHA"
