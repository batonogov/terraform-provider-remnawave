#!/usr/bin/env bash
set -euo pipefail

# Asserts that a CI run's job list is complete and entirely successful.
#
# Extracted so the release gate can apply the identical standard twice: once to
# the CI run that triggered it, and once to the run belonging to the revision a
# release actually targets. Those are the same question asked about different
# commits, and answering it two different ways is how a gate drifts.
#
# Inputs:
#   CI_JOBS_FILE - JSON from actions/runs/<id>/jobs

fail() {
  echo "release gate: $*" >&2
  exit 1
}

: "${CI_JOBS_FILE:?CI_JOBS_FILE is required}"

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
compat_versions="$repository_dir/compat-versions.json"

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
  "Release Artifact Tests"
  "Release Supply Chain Tests"
  "Repository Policy Tests"
  "Vulnerability Scan"
)

while IFS= read -r label; do
  required_jobs+=("Acceptance Tests ($label)")
done < <(jq -r '.versions[] | select(.supported) | .version | ltrimstr("v")' "$compat_versions")

for required_job in "${required_jobs[@]}"; do
  matches=$(jq --arg name "$required_job" '
    [.jobs[] | select(.name == $name and .status == "completed" and .conclusion == "success")]
    | length
  ' "$CI_JOBS_FILE")
  [[ "$matches" == "1" ]] ||
    fail "required job $required_job has $matches successful results, want exactly 1"
done
