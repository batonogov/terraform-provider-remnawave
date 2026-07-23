#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/configure-repository-security.sh --check
  RELEASE_REVIEWER_ID=<github-user-or-team-id> \
    scripts/configure-repository-security.sh --apply

REPOSITORY may override the current owner/name detected by gh.
EOF
}

fail() {
  echo "repository security: $*" >&2
  exit 1
}

[[ $# == 1 ]] || {
  usage >&2
  exit 2
}
mode=$1
[[ "$mode" == "--check" || "$mode" == "--apply" ]] || {
  usage >&2
  exit 2
}

command -v gh >/dev/null || fail "gh is required"
command -v jq >/dev/null || fail "jq is required"
gh auth status >/dev/null 2>&1 || fail "gh authentication is required"

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
settings_dir="$repository_dir/.github/repository-settings"
repository=${REPOSITORY:-$(gh repo view --json nameWithOwner --jq .nameWithOwner)}
api_version=2026-03-10

for policy in \
  main-ruleset.json \
  release-tags-ruleset.json \
  actions-permissions.json \
  workflow-permissions.json \
  release-environment.json; do
  jq -e . "$settings_dir/$policy" >/dev/null ||
    fail "$policy is not valid JSON"
done

api() {
  gh api -H "X-GitHub-Api-Version: $api_version" "$@"
}

ruleset_id() {
  local name=$1
  api "repos/$repository/rulesets" \
    --jq ".[] | select(.name == \"$name\") | .id" |
    head -n 1
}

upsert_ruleset() {
  local policy=$1
  local name id
  name=$(jq -r .name "$policy")
  id=$(ruleset_id "$name")
  if [[ -n "$id" ]]; then
    api --method PUT "repos/$repository/rulesets/$id" --input "$policy" >/dev/null
    echo "repository security: updated ruleset $name"
  else
    api --method POST "repos/$repository/rulesets" --input "$policy" >/dev/null
    echo "repository security: created ruleset $name"
  fi
}

if [[ "$mode" == "--apply" ]]; then
  : "${RELEASE_REVIEWER_ID:?RELEASE_REVIEWER_ID is required for --apply}"
  [[ "$RELEASE_REVIEWER_ID" =~ ^[0-9]+$ ]] ||
    fail "RELEASE_REVIEWER_ID must be a numeric GitHub user or team ID"

  upsert_ruleset "$settings_dir/main-ruleset.json"
  upsert_ruleset "$settings_dir/release-tags-ruleset.json"
  api --method PUT "repos/$repository/actions/permissions" \
    --input "$settings_dir/actions-permissions.json" >/dev/null
  api --method PUT "repos/$repository/actions/permissions/workflow" \
    --input "$settings_dir/workflow-permissions.json" >/dev/null

  environment_payload=$(mktemp "${TMPDIR:-/tmp}/release-environment.XXXXXX")
  trap 'rm -f "$environment_payload"' EXIT
  jq --argjson reviewer_id "$RELEASE_REVIEWER_ID" \
    '.reviewers = [{"type": "User", "id": $reviewer_id}]' \
    "$settings_dir/release-environment.json" >"$environment_payload"
  api --method PUT "repos/$repository/environments/release" \
    --input "$environment_payload" >/dev/null
  rm -f "$environment_payload"
  trap - EXIT
  echo "repository security: configured protected release environment"
fi

temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/repository-security.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

main_id=$(ruleset_id main)
[[ -n "$main_id" ]] || fail "main ruleset is missing"
api "repos/$repository/rulesets/$main_id" >"$temporary_dir/main.json"

tag_id=$(ruleset_id release-tags)
[[ -n "$tag_id" ]] || fail "release-tags ruleset is missing"
api "repos/$repository/rulesets/$tag_id" >"$temporary_dir/tags.json"

expected_main_types=$(jq -c '[.rules[].type] | sort' "$settings_dir/main-ruleset.json")
actual_main_types=$(jq -c '[.rules[].type] | sort' "$temporary_dir/main.json")
[[ "$actual_main_types" == "$expected_main_types" ]] ||
  fail "main ruleset types differ from policy"

expected_checks=$(jq -c '
  [.rules[]
    | select(.type == "required_status_checks")
    | .parameters.required_status_checks[].context]
  | sort
' "$settings_dir/main-ruleset.json")
actual_checks=$(jq -c '
  [.rules[]
    | select(.type == "required_status_checks")
    | .parameters.required_status_checks[].context]
  | sort
' "$temporary_dir/main.json")
[[ "$actual_checks" == "$expected_checks" ]] ||
  fail "main required checks differ from policy"

jq -e '
  .enforcement == "active" and
  any(.rules[];
    .type == "pull_request" and
    .parameters.required_approving_review_count >= 1 and
    .parameters.dismiss_stale_reviews_on_push == true and
    .parameters.require_last_push_approval == true and
    .parameters.required_review_thread_resolution == true)
' "$temporary_dir/main.json" >/dev/null ||
  fail "main pull-request review policy is incomplete"

jq -e '
  .enforcement == "active" and
  (.conditions.ref_name.include | index("refs/tags/v*") != null) and
  (["deletion", "non_fast_forward", "update"] -
    [.rules[].type] | length == 0)
' "$temporary_dir/tags.json" >/dev/null ||
  fail "release tag ruleset is incomplete"

api "repos/$repository/actions/permissions" >"$temporary_dir/actions.json"
jq -e '.enabled == true and .sha_pinning_required == true' \
  "$temporary_dir/actions.json" >/dev/null ||
  fail "full-SHA action pinning is not enforced"

api "repos/$repository/actions/permissions/workflow" >"$temporary_dir/workflow.json"
jq -e '
  .default_workflow_permissions == "read" and
  .can_approve_pull_request_reviews == false
' "$temporary_dir/workflow.json" >/dev/null ||
  fail "default workflow permissions are not read-only"

api "repos/$repository/environments/release" >"$temporary_dir/environment.json"
jq -e '
  any(.protection_rules[];
    .type == "required_reviewers" and
    .prevent_self_review == true and
    (.reviewers | length) >= 1)
' "$temporary_dir/environment.json" >/dev/null ||
  fail "release environment does not require an independent reviewer"

echo "repository security: enforced settings match the committed policy"
echo "repository security: confirm release immutability in GitHub Settings > General > Releases"
