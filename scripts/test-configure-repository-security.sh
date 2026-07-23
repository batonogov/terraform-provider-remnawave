#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/repository-policy-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

mkdir -p "$temporary_dir/bin" "$temporary_dir/fixtures"
cp "$repository_dir/.github/repository-settings/main-ruleset.json" \
  "$temporary_dir/fixtures/main.json"
cp "$repository_dir/.github/repository-settings/release-tags-ruleset.json" \
  "$temporary_dir/fixtures/tags.json"
cp "$repository_dir/.github/repository-settings/actions-permissions.json" \
  "$temporary_dir/fixtures/actions.json"
cp "$repository_dir/.github/repository-settings/workflow-permissions.json" \
  "$temporary_dir/fixtures/workflow.json"
jq '{
  protection_rules: [{
    type: "required_reviewers",
    prevent_self_review: .prevent_self_review,
    reviewers: [{type: "User", reviewer: {id: 42, login: "reviewer"}}]
  }],
  deployment_branch_policy: .deployment_branch_policy
}' "$repository_dir/.github/repository-settings/release-environment.json" \
  >"$temporary_dir/fixtures/environment.json"

cp "$script_dir/testdata/mock-gh-repository-security" "$temporary_dir/bin/gh"
chmod +x "$temporary_dir/bin/gh"

run_policy() {
  PATH="$temporary_dir/bin:$PATH" \
    MOCK_FIXTURE_DIR="$temporary_dir/fixtures" \
    MOCK_API_LOG="$temporary_dir/api.log" \
    REPOSITORY=owner/repository \
    "$script_dir/configure-repository-security.sh" "$@"
}

run_policy --check >/dev/null
RELEASE_REVIEWER_ID=42 run_policy --apply >/dev/null

for expected in \
  "PUT repos/owner/repository/rulesets/1" \
  "PUT repos/owner/repository/rulesets/2" \
  "PUT repos/owner/repository/actions/permissions" \
  "PUT repos/owner/repository/actions/permissions/workflow" \
  "PUT repos/owner/repository/environments/release"; do
  grep -Fx "$expected" "$temporary_dir/api.log" >/dev/null || {
    echo "missing mocked API mutation: $expected" >&2
    exit 1
  }
done

jq '(.rules[]
  | select(.type == "required_status_checks")
  | .parameters.required_status_checks) |=
  map(select(.context != "Unit Tests"))' \
  "$repository_dir/.github/repository-settings/main-ruleset.json" \
  >"$temporary_dir/fixtures/main.json"
if run_policy --check >/dev/null 2>&1; then
  echo "policy audit unexpectedly accepted a missing required check" >&2
  exit 1
fi

echo "repository security policy tests passed"
