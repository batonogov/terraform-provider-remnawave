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
  can_admins_bypass: false,
  protection_rules: [{
    type: "required_reviewers",
    prevent_self_review: .prevent_self_review,
    reviewers: [{type: "User", reviewer: {id: 42, login: "reviewer"}}]
  }],
  deployment_branch_policy: .deployment_branch_policy
}' "$repository_dir/.github/repository-settings/release-environment.json" \
  >"$temporary_dir/fixtures/environment.json"
printf '%s\n' '{"enabled":true}' \
  >"$temporary_dir/fixtures/immutable-releases.json"
printf '%s\n' \
  '{"secrets":[{"name":"REPOSITORY_SECURITY_AUDIT_TOKEN"}]}' \
  >"$temporary_dir/fixtures/repository-secrets.json"
printf '%s\n' \
  '{"secrets":[{"name":"RELEASE_GPG_PRIVATE_KEY"},{"name":"RELEASE_GPG_PASSPHRASE"}]}' \
  >"$temporary_dir/fixtures/environment-secrets.json"

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

jq '
  (.rules[]
    | select(.type == "pull_request")
    | .parameters.required_approving_review_count) = 0
  | (.rules[]
    | select(.type == "pull_request")
    | .parameters.require_last_push_approval) = false
' "$repository_dir/.github/repository-settings/main-ruleset.json" \
  >"$temporary_dir/fixtures/main.json"
jq '{
  can_admins_bypass: true,
  protection_rules: [],
  deployment_branch_policy: .deployment_branch_policy
}' "$repository_dir/.github/repository-settings/release-environment.json" \
  >"$temporary_dir/fixtures/environment.json"

run_policy --check-solo >/dev/null
run_policy --apply-solo >/dev/null

printf '%s\n' \
  '{"secrets":[{"name":"RELEASE_GPG_PRIVATE_KEY"}]}' \
  >"$temporary_dir/fixtures/repository-secrets.json"
if run_policy --check-solo >/dev/null 2>&1; then
  echo "policy audit unexpectedly accepted a repository-level release secret" >&2
  exit 1
fi
printf '%s\n' \
  '{"secrets":[{"name":"REPOSITORY_SECURITY_AUDIT_TOKEN"}]}' \
  >"$temporary_dir/fixtures/repository-secrets.json"

jq '(.rules[]
  | select(.type == "required_status_checks")
  | .parameters.required_status_checks) |=
  map(select(.context != "Unit Tests"))' \
  "$temporary_dir/fixtures/main.json" \
  >"$temporary_dir/fixtures/main-with-missing-check.json"
mv "$temporary_dir/fixtures/main-with-missing-check.json" \
  "$temporary_dir/fixtures/main.json"
if run_policy --check-solo >/dev/null 2>&1; then
  echo "policy audit unexpectedly accepted a missing required check" >&2
  exit 1
fi

# The branch ruleset and the release gate must demand the same CI jobs. The
# gate derives its acceptance matrix from compat-versions.json, while the
# ruleset is static JSON, so a newly supported version silently becomes
# non-blocking: a pull request could merge with that version red, land it on
# main, and stall the release the gate then refuses to pass.
missing_contexts=$(
  jq -r --slurpfile compat "$repository_dir/compat-versions.json" '
    [$compat[0].versions[] | select(.supported)
      | "Acceptance Tests (\(.version | ltrimstr("v")))"] as $expected
    | [.rules[] | select(.type == "required_status_checks")
      | .parameters.required_status_checks[].context] as $declared
    | $expected - $declared
    | .[]
  ' "$repository_dir/.github/repository-settings/main-ruleset.json"
)
if [ -n "$missing_contexts" ]; then
  echo "main ruleset does not require every supported acceptance job:" >&2
  printf '%s\n' "$missing_contexts" | sed 's/^/  /' >&2
  exit 1
fi

echo "repository security policy tests passed"
