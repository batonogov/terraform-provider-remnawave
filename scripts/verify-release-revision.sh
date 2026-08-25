#!/usr/bin/env bash
set -euo pipefail

# Selects the CI run that proves a release target revision was tested.
#
# The gate used to assert that the release target equals the newest CI-validated
# commit on main. release-please, however, always tags the merge commit of its
# own release pull request. The two coincide only when CI passes on that exact
# commit and nothing else merges in the meantime, so a single flaky run stranded
# the release permanently: main moved on, and the equality could never hold
# again. 1.7.1 had to be re-cut by hand for exactly this reason.
#
# What the gate actually needs is not "the release is the newest commit" but
# "the release target passed CI". This script asserts the latter directly, which
# is the same guarantee without the fragility: re-running a flaked job now
# recovers the release instead of stranding it.
#
# The guarantee is preserved by requiring the proving run to be a push build of
# main for this very revision, and by requiring the revision to be contained in
# main. Without those, a green run from a pull-request branch would qualify.
#
# Inputs:
#   RELEASE_SHA          - commit the release targets
#   RELEASE_RUNS_FILE    - JSON from actions/runs?head_sha=<RELEASE_SHA>
#   RELEASE_SHA_ON_MAIN  - "true" when RELEASE_SHA is contained in origin/main
#   CI_WORKFLOW_NAME     - workflow name that constitutes CI (default "CI")
#
# Prints the id of the qualifying run, and appends release_run_id to
# GITHUB_OUTPUT when that is set.

fail() {
  echo "release gate: $*" >&2
  exit 1
}

: "${RELEASE_SHA:?RELEASE_SHA is required}"
: "${RELEASE_RUNS_FILE:?RELEASE_RUNS_FILE is required}"
: "${RELEASE_SHA_ON_MAIN:?RELEASE_SHA_ON_MAIN is required}"
ci_workflow_name=${CI_WORKFLOW_NAME:-CI}

[[ "$RELEASE_SHA" =~ ^[0-9a-f]{40}$ ]] ||
  fail "release SHA is not a full commit SHA"

[[ "$RELEASE_SHA_ON_MAIN" == "true" ]] ||
  fail "release SHA $RELEASE_SHA is not contained in origin/main"

jq -e '.workflow_runs | type == "array"' "$RELEASE_RUNS_FILE" >/dev/null ||
  fail "workflow runs response is missing or malformed"

# A run only proves anything about this revision if it is a completed,
# successful push build of main for this exact SHA. Anything looser would admit
# a green pull-request run for the same tree.
qualifying=$(jq -r \
  --arg sha "$RELEASE_SHA" \
  --arg workflow "$ci_workflow_name" '
    [ .workflow_runs[]
      | select(
          .name == $workflow and
          .event == "push" and
          .head_branch == "main" and
          .head_sha == $sha and
          .status == "completed" and
          .conclusion == "success"
        )
    ]
    | sort_by(.run_number)
    | reverse
    | .[0].id // empty
  ' "$RELEASE_RUNS_FILE")

if [[ -z "$qualifying" ]]; then
  observed=$(jq -r \
    --arg workflow "$ci_workflow_name" '
      [ .workflow_runs[]
        | select(.name == $workflow)
        | "\(.event)/\(.head_branch)/\(.status)/\(.conclusion // "none")"
      ]
      | unique
      | join("; ")
    ' "$RELEASE_RUNS_FILE")
  fail "no successful $ci_workflow_name push build of main for $RELEASE_SHA (observed: ${observed:-none})"
fi

[[ "$qualifying" =~ ^[1-9][0-9]*$ ]] ||
  fail "selected run id $qualifying is not a positive integer"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "release_run_id=$qualifying" >>"$GITHUB_OUTPUT"
fi
printf '%s\n' "$qualifying"
