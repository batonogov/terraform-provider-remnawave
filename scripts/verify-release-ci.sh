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

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
CI_JOBS_FILE="$CI_JOBS_FILE" "$script_dir/verify-ci-jobs.sh"

echo "release gate: CI run validated for $CI_RUN_HEAD_SHA"
