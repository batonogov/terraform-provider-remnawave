#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-worktree-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT
fixture_repo="$temporary_dir/repository"

git init -q "$fixture_repo"
(
  cd "$fixture_repo"
  git config user.name "Release Test"
  git config user.email "release-test@example.com"
  printf 'tracked\n' >fixture.txt
  printf '/dist/\n' >.gitignore
  git add fixture.txt .gitignore
  git -c commit.gpgsign=false commit -qm "test fixture"
  git tag v1.2.3
  git checkout -q --detach v1.2.3
)

fixture_sha=$(git -C "$fixture_repo" rev-parse HEAD)

run_check() {
  (
    cd "$fixture_repo"
    "$script_dir/verify-release-worktree.sh" "${EXPECTED_SHA:-$fixture_sha}" "${TAG_NAME:-v1.2.3}"
  )
}

expect_failure() {
  local name=$1
  if run_check >/dev/null 2>&1; then
    echo "$name: worktree verification unexpectedly succeeded" >&2
    exit 1
  fi
}

run_check >/dev/null

(
  cd "$fixture_repo"
  mkdir -p dist
  printf 'ignored\n' >dist/generated.txt
)
run_check >/dev/null

EXPECTED_SHA=0000000000000000000000000000000000000000 expect_failure "wrong revision"

printf 'changed\n' >>"$fixture_repo/fixture.txt"
expect_failure "tracked modification"
git -C "$fixture_repo" checkout -- fixture.txt

printf 'unexpected\n' >"$fixture_repo/untracked.txt"
expect_failure "unexpected untracked file"
rm "$fixture_repo/untracked.txt"

git -C "$fixture_repo" switch -q --detach
git -C "$fixture_repo" switch -q -c fixture-branch
expect_failure "attached branch"

echo "release worktree tests passed"
