#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "release worktree: $*" >&2
  exit 1
}

[[ $# -eq 2 ]] ||
  fail "usage: verify-release-worktree.sh EXPECTED_SHA TAG_NAME"

expected_sha=$1
tag_name=$2

[[ "$expected_sha" =~ ^[0-9a-f]{40}$ ]] ||
  fail "expected SHA must be a full lowercase commit SHA"
[[ "$tag_name" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] ||
  fail "tag name is not strict semantic versioning"

head_sha=$(git rev-parse HEAD)
tag_sha=$(git rev-list -n 1 "$tag_name")
[[ "$head_sha" == "$expected_sha" ]] ||
  fail "HEAD $head_sha does not match expected SHA $expected_sha"
[[ "$tag_sha" == "$expected_sha" ]] ||
  fail "tag $tag_name targets $tag_sha, not $expected_sha"

if git symbolic-ref -q HEAD >/dev/null; then
  fail "release checkout must be detached"
fi

tracked_changes=$(git status --porcelain=v1 --untracked-files=no)
[[ -z "$tracked_changes" ]] ||
  fail "tracked or staged changes are present"

unexpected_files=$(git ls-files --others --exclude-standard)
[[ -z "$unexpected_files" ]] ||
  fail "unexpected untracked files are present: ${unexpected_files//$'\n'/, }"

echo "release worktree: clean detached checkout verified at $expected_sha"
