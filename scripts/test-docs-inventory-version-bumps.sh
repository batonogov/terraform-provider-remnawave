#!/usr/bin/env bash
set -euo pipefail

# Replays release-please version bumps against the documentation inventory
# invariant.
#
# The provider version constraint in the copyable required_providers snippet is
# written by release-please and asserted by check-docs-inventory.sh. Those are
# two independent mechanisms operating on one string, so they can silently
# disagree. They did: the constraint was annotated `x-release-please-version`
# (which rewrites the full version) while the checker demanded a fixed patch
# component, so every patch release produced a red main and a stuck release.
# The disagreement was invisible on ordinary pull requests because the bumped
# version only exists on the release commit.
#
# This test makes the release commit reachable from CI by applying the
# updater's own replacement semantics to a scratch copy of the repository and
# running the real checker over the result.

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/docs-inventory-bump-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

annotated_files=(
  README.md
  docs/index.md
  examples/getting-started/main.tf
  examples/provider.tf
  examples/provider/provider.tf
)

fail() {
  echo "docs inventory bump tests: $*" >&2
  exit 1
}

# Mirrors src/updaters/generic.ts from release-please. The `major` scope
# replaces the first MAJOR_VERSION_REGEX (/\d+\b/) match on an annotated line;
# the `version` scope replaces the first VERSION_REGEX (full semver) match.
# Both replace a single occurrence per line.
apply_generic_updater() {
  local version=$1 root=$2 file major
  major=${version%%.*}
  for file in "${annotated_files[@]}"; do
    perl -pi \
      -e 'if (/x-release-please-major/) { s/\d+\b/'"$major"'/ }' \
      -e 'elsif (/x-release-please-version/) {' \
      -e '  s/\d+\.\d+\.\d+(-[\w.]+)?(\+[-\w.]+)?/'"$version"'/ }' \
      "$root/$file"
  done
}

set_manifest_version() {
  local version=$1 root=$2
  printf '{\n  ".": "%s"\n}\n' "$version" >"$root/.release-please-manifest.json"
}

# A scratch copy so the checker runs against a real tree: it also verifies
# resource counts, docs, and examples, none of which this test perturbs.
new_worktree() {
  local name=$1
  local root="$temporary_dir/$name"
  mkdir -p "$root"
  tar -cf - -C "$repository_dir" \
    --exclude .git --exclude dist . | tar -xf - -C "$root"
  printf '%s' "$root"
}

# Simulates a release: bump the manifest, let the updater rewrite the
# annotated lines, then assert the checker agrees with what the updater wrote.
expect_bump_accepted() {
  local name=$1 version=$2 root
  root=$(new_worktree "accept-$name")
  set_manifest_version "$version" "$root"
  apply_generic_updater "$version" "$root"
  if ! "$root/scripts/check-docs-inventory.sh" >/dev/null 2>&1; then
    "$root/scripts/check-docs-inventory.sh" >/dev/null || true
    fail "$name bump to $version was rejected by the inventory check"
  fi
}

expect_constraint() {
  local name=$1 version=$2 expected=$3 root file
  root=$(new_worktree "constraint-$name")
  set_manifest_version "$version" "$root"
  apply_generic_updater "$version" "$root"
  for file in "${annotated_files[@]}"; do
    grep -Fq -- "version = \"$expected\"" "$root/$file" ||
      fail "$name bump to $version: $file does not carry $expected"
  done
}

# The regression itself. With the full-version annotation the updater writes a
# moving patch component, which the invariant must reject — if this stops
# failing, the checker has been weakened and the trap is back.
expect_full_version_annotation_rejected() {
  local root
  root=$(new_worktree "legacy-annotation")
  perl -pi -e 's/# x-release-please-major/# x-release-please-version/' \
    "$root/README.md"
  set_manifest_version 1.7.2 "$root"
  apply_generic_updater 1.7.2 "$root"
  if "$root/scripts/check-docs-inventory.sh" >/dev/null 2>&1; then
    fail "the full-version annotation must not satisfy the inventory check"
  fi
}

current_version=$(awk -F'"' '/"\."/ { print $4; exit }' \
  "$repository_dir/.release-please-manifest.json")
current_major=${current_version%%.*}

# The checked-in tree must already satisfy the invariant.
"$repository_dir/scripts/check-docs-inventory.sh" >/dev/null ||
  fail "the working tree does not satisfy the inventory check"

# Patch and minor bumps must not move the constraint; a major bump must.
expect_bump_accepted patch "$current_major.7.2"
expect_bump_accepted minor "$current_major.8.0"
expect_bump_accepted major "$((current_major + 1)).0.0"

expect_constraint patch "$current_major.7.2" "~> $current_major.0"
expect_constraint minor "$current_major.8.0" "~> $current_major.0"
expect_constraint major "$((current_major + 1)).0.0" \
  "~> $((current_major + 1)).0"

expect_full_version_annotation_rejected

echo "docs inventory bump tests passed"
