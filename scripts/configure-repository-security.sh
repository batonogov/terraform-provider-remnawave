#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/configure-repository-security.sh --check
  scripts/configure-repository-security.sh --check-solo
  RELEASE_REVIEWER_ID=<github-user-or-team-id> \
    scripts/configure-repository-security.sh --apply
  scripts/configure-repository-security.sh --apply-solo

REPOSITORY may override the current owner/name detected by gh.

The solo modes enforce pull requests, required checks, signed commits, tag
protection, Actions restrictions, and release-Environment scoping without an
approval requirement. Use them only while no second trusted reviewer exists.
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
case "$mode" in
  --check)
    action=check
    review_mode=strict
    ;;
  --apply)
    action=apply
    review_mode=strict
    ;;
  --check-solo)
    action=check
    review_mode=solo
    ;;
  --apply-solo)
    action=apply
    review_mode=solo
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
check_option=--check
if [[ "$review_mode" == "solo" ]]; then
  check_option=--check-solo
fi

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

temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/repository-security.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

main_policy="$settings_dir/main-ruleset.json"
environment_policy="$settings_dir/release-environment.json"
if [[ "$review_mode" == "solo" ]]; then
  main_policy="$temporary_dir/main-policy.json"
  jq '
    (.rules[]
      | select(.type == "pull_request")
      | .parameters.required_approving_review_count) = 0
    | (.rules[]
      | select(.type == "pull_request")
      | .parameters.require_last_push_approval) = false
  ' "$settings_dir/main-ruleset.json" >"$main_policy"

  environment_policy="$temporary_dir/release-environment-policy.json"
  jq '
    .prevent_self_review = false
    | .reviewers = []
  ' "$settings_dir/release-environment.json" >"$environment_policy"
fi

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

if [[ "$action" == "apply" ]]; then
  environment_payload="$environment_policy"
  if [[ "$review_mode" == "strict" ]]; then
    : "${RELEASE_REVIEWER_ID:?RELEASE_REVIEWER_ID is required for --apply}"
    [[ "$RELEASE_REVIEWER_ID" =~ ^[0-9]+$ ]] ||
      fail "RELEASE_REVIEWER_ID must be a numeric GitHub user or team ID"

    environment_payload="$temporary_dir/release-environment-apply.json"
    jq --argjson reviewer_id "$RELEASE_REVIEWER_ID" \
      '.reviewers = [{"type": "User", "id": $reviewer_id}]' \
      "$environment_policy" >"$environment_payload"
  else
    echo \
      "repository security: WARNING: independent reviews remain deferred in solo mode" \
      >&2
  fi

  upsert_ruleset "$main_policy"
  upsert_ruleset "$settings_dir/release-tags-ruleset.json"
  api --method PUT "repos/$repository/actions/permissions" \
    --input "$settings_dir/actions-permissions.json" >/dev/null
  api --method PUT "repos/$repository/actions/permissions/workflow" \
    --input "$settings_dir/workflow-permissions.json" >/dev/null

  # Dependabot version updates run on a weekly schedule. Alerts and their
  # automated security fixes are the out-of-band channel that opens a pull
  # request the moment a CVE lands in a dependency, instead of waiting for the
  # next Monday. Both were off while the weekly updates ran, which reads as
  # "dependencies are watched" without the part that reacts to disclosures.
  api --method PUT "repos/$repository/vulnerability-alerts" >/dev/null
  api --method PUT "repos/$repository/automated-security-fixes" >/dev/null

  api --method PUT "repos/$repository/environments/release" \
    --input "$environment_payload" >/dev/null
  echo "repository security: configured release environment"
fi

main_id=$(ruleset_id main)
[[ -n "$main_id" ]] || fail "main ruleset is missing"
api "repos/$repository/rulesets/$main_id" >"$temporary_dir/main.json"

tag_id=$(ruleset_id release-tags)
[[ -n "$tag_id" ]] || fail "release-tags ruleset is missing"
api "repos/$repository/rulesets/$tag_id" >"$temporary_dir/tags.json"

expected_main_types=$(jq -c '[.rules[].type] | sort' "$main_policy")
actual_main_types=$(jq -c '[.rules[].type] | sort' "$temporary_dir/main.json")
[[ "$actual_main_types" == "$expected_main_types" ]] ||
  fail "main ruleset types differ from policy"

expected_checks=$(jq -c '
  [.rules[]
    | select(.type == "required_status_checks")
    | .parameters.required_status_checks[]
    | {context, integration_id}]
  | sort_by(.context)
' "$main_policy")
actual_checks=$(jq -c '
  [.rules[]
    | select(.type == "required_status_checks")
    | .parameters.required_status_checks[]
    | {context, integration_id}]
  | sort_by(.context)
' "$temporary_dir/main.json")
[[ "$actual_checks" == "$expected_checks" ]] ||
  fail "main required checks differ from policy"

if [[ "$review_mode" == "strict" ]]; then
  jq -e '
    .enforcement == "active" and
    (.bypass_actors | length) == 0 and
    any(.rules[];
      .type == "pull_request" and
      .parameters.required_approving_review_count >= 1 and
      .parameters.dismiss_stale_reviews_on_push == true and
      .parameters.require_last_push_approval == true and
      .parameters.required_review_thread_resolution == true)
  ' "$temporary_dir/main.json" >/dev/null ||
    fail "main pull-request review policy is incomplete"
else
  jq -e '
    .enforcement == "active" and
    (.bypass_actors | length) == 0 and
    any(.rules[];
      .type == "pull_request" and
      .parameters.required_approving_review_count == 0 and
      .parameters.require_last_push_approval == false and
      .parameters.required_review_thread_resolution == true)
  ' "$temporary_dir/main.json" >/dev/null ||
    fail "main solo-maintainer pull-request policy is incomplete"
fi

jq -e '
  .enforcement == "active" and
  (.bypass_actors | length) == 0 and
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
# Compare against the committed policy rather than a second, hardcoded copy of
# the expectation. release-please must be allowed to open its release pull
# request, and a literal here would silently contradict the file the apply path
# writes.
jq -e --slurpfile expected "$settings_dir/workflow-permissions.json" '
  .default_workflow_permissions == $expected[0].default_workflow_permissions and
  .can_approve_pull_request_reviews == $expected[0].can_approve_pull_request_reviews
' "$temporary_dir/workflow.json" >/dev/null ||
  fail "workflow permissions do not match the committed policy"

api "repos/$repository/environments/release" >"$temporary_dir/environment.json"
jq -e '
  .deployment_branch_policy.protected_branches == true and
  .deployment_branch_policy.custom_branch_policies == false
' "$temporary_dir/environment.json" >/dev/null ||
  fail "release environment deployment policy is incomplete"

if [[ "$review_mode" == "strict" ]]; then
  jq -e '
    any(.protection_rules[];
      .type == "required_reviewers" and
      .prevent_self_review == true and
      (.reviewers | length) >= 1)
  ' "$temporary_dir/environment.json" >/dev/null ||
    fail "release environment does not require independent review"
  if [[ "$action" == "check" ]]; then
    jq -e '.can_admins_bypass == false' \
      "$temporary_dir/environment.json" >/dev/null ||
      fail "release environment still permits administrator bypass"
  elif jq -e '.can_admins_bypass == true' \
    "$temporary_dir/environment.json" >/dev/null; then
    echo \
      "repository security: disable release Environment administrator bypass, then run $0 $check_option" \
      >&2
  fi
else
  jq -e '
    [.protection_rules[] | select(.type == "required_reviewers")]
    | length == 0
  ' "$temporary_dir/environment.json" >/dev/null ||
    fail "release environment unexpectedly has a partial reviewer policy"
  if jq -e '.can_admins_bypass == true' "$temporary_dir/environment.json" >/dev/null; then
    echo \
      "repository security: WARNING: release admin bypass remains enabled in solo mode" \
      >&2
  fi
fi

api "repos/$repository/immutable-releases" >"$temporary_dir/immutable-releases.json"
jq -e '.enabled == true' "$temporary_dir/immutable-releases.json" >/dev/null ||
  fail "release immutability is not enabled"

# The alerts endpoint answers 204 when enabled and 404 when not, so the exit
# status is the whole answer.
api "repos/$repository/vulnerability-alerts" >/dev/null 2>&1 ||
  fail "Dependabot vulnerability alerts are not enabled"

api "repos/$repository/automated-security-fixes" \
  >"$temporary_dir/automated-security-fixes.json"
jq -e '.enabled == true and .paused == false' \
  "$temporary_dir/automated-security-fixes.json" >/dev/null ||
  fail "Dependabot automated security fixes are not enabled"

if [[ "$action" == "check" ]]; then
  api "repos/$repository/actions/secrets" >"$temporary_dir/repository-secrets.json"
  api "repos/$repository/environments/release/secrets" \
    >"$temporary_dir/environment-secrets.json"
  for secret_name in RELEASE_GPG_PRIVATE_KEY RELEASE_GPG_PASSPHRASE; do
    if jq -e --arg name "$secret_name" \
      '.secrets | any(.name == $name)' \
      "$temporary_dir/repository-secrets.json" >/dev/null; then
      fail "$secret_name must not exist as a repository-level secret"
    fi
    jq -e --arg name "$secret_name" \
      '.secrets | any(.name == $name)' \
      "$temporary_dir/environment-secrets.json" >/dev/null ||
      fail "$secret_name is missing from the release environment"
  done
else
  echo \
    "repository security: migrate release GPG secrets, then rerun $0 $check_option" \
    >&2
fi

echo "repository security: enforced settings match the committed $review_mode policy"
