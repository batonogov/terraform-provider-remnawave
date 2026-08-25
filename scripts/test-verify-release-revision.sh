#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-revision-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

release_sha=1111111111111111111111111111111111111111
other_sha=2222222222222222222222222222222222222222

make_runs() {
  local file=$1
  shift
  jq -n --argjson runs "$1" '{workflow_runs: $runs}' >"$file"
}

good_run() {
  jq -n --arg sha "$release_sha" '{
    id: 4242,
    run_number: 7,
    name: "CI",
    event: "push",
    head_branch: "main",
    head_sha: $sha,
    status: "completed",
    conclusion: "success"
  }'
}

run_check() {
  (
    RELEASE_SHA="${RELEASE_SHA:-$release_sha}" \
      RELEASE_RUNS_FILE="${RELEASE_RUNS_FILE:-$temporary_dir/runs.json}" \
      RELEASE_SHA_ON_MAIN="${RELEASE_SHA_ON_MAIN:-true}" \
      "$script_dir/verify-release-revision.sh"
  )
}

expect_rejected() {
  local name=$1
  shift
  if (export "$@"; run_check) >/dev/null 2>&1; then
    echo "$name: gate unexpectedly accepted the revision" >&2
    exit 1
  fi
}

make_runs "$temporary_dir/runs.json" "[$(good_run)]"

selected=$(run_check)
[[ "$selected" == "4242" ]] || {
  echo "expected run 4242, got $selected" >&2
  exit 1
}

# The whole point of the change: a revision that is no longer main's tip still
# releases, provided its own CI run is green.
make_runs "$temporary_dir/stale.json" "[$(good_run)]"
selected=$(RELEASE_RUNS_FILE="$temporary_dir/stale.json" run_check)
[[ "$selected" == "4242" ]] || {
  echo "a superseded but CI-green revision must still qualify" >&2
  exit 1
}

# A revision that never passed CI must not release, however old or new it is.
make_runs "$temporary_dir/failed.json" \
  "[$(good_run | jq '.conclusion = "failure"')]"
expect_rejected "failed run" RELEASE_RUNS_FILE="$temporary_dir/failed.json"

make_runs "$temporary_dir/running.json" \
  "[$(good_run | jq '.status = "in_progress" | .conclusion = null')]"
expect_rejected "incomplete run" RELEASE_RUNS_FILE="$temporary_dir/running.json"

# A green run from a pull-request branch proves nothing about main.
make_runs "$temporary_dir/pr.json" \
  "[$(good_run | jq '.event = "pull_request"')]"
expect_rejected "pull request run" RELEASE_RUNS_FILE="$temporary_dir/pr.json"

make_runs "$temporary_dir/branch.json" \
  "[$(good_run | jq '.head_branch = "topic"')]"
expect_rejected "non-main branch" RELEASE_RUNS_FILE="$temporary_dir/branch.json"

# The run must belong to this revision, not merely be adjacent to it.
make_runs "$temporary_dir/othersha.json" \
  "[$(good_run | jq --arg sha "$other_sha" '.head_sha = $sha')]"
expect_rejected "different revision" RELEASE_RUNS_FILE="$temporary_dir/othersha.json"

# Some other successful workflow is not CI.
make_runs "$temporary_dir/otherworkflow.json" \
  "[$(good_run | jq '.name = "Release Please"')]"
expect_rejected "unrelated workflow" \
  RELEASE_RUNS_FILE="$temporary_dir/otherworkflow.json"

make_runs "$temporary_dir/empty.json" "[]"
expect_rejected "no runs" RELEASE_RUNS_FILE="$temporary_dir/empty.json"

# Containment in main is checked by the caller; the gate must honour a negative.
expect_rejected "revision not on main" RELEASE_SHA_ON_MAIN=false
expect_rejected "malformed sha" RELEASE_SHA=abc123

# Among several green push builds of the same revision, take the newest.
make_runs "$temporary_dir/multiple.json" \
  "[$(good_run), $(good_run | jq '.id = 9001 | .run_number = 9')]"
selected=$(RELEASE_RUNS_FILE="$temporary_dir/multiple.json" run_check)
[[ "$selected" == "9001" ]] || {
  echo "expected newest run 9001, got $selected" >&2
  exit 1
}

echo "release revision tests passed"
