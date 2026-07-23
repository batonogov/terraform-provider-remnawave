#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "release artifacts: $*" >&2
  exit 1
}

[[ $# -eq 4 ]] ||
  fail "usage: verify-release-artifacts.sh DIST_DIR VERSION EXPECTED_SHA EXPECTED_MODULE_VERSION"

dist_dir=$1
release_version=$2
expected_sha=$3
expected_module_version=$4
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
targets_file="$repository_dir/release-targets.json"

[[ -d "$dist_dir" ]] || fail "distribution directory does not exist: $dist_dir"
[[ "$release_version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] ||
  fail "release version is not strict semantic versioning"
[[ "$expected_sha" =~ ^[0-9a-f]{40}$ ]] ||
  fail "expected SHA must be a full lowercase commit SHA"
[[ "$expected_module_version" == "v$release_version" ]] ||
  fail "expected module version must be v$release_version"

jq -e '
  (.project_name | type == "string" and length > 0)
  and (.module_path | type == "string" and length > 0)
  and (.targets | type == "array" and length > 0)
  and ([.targets[]
    | (.goos | type == "string" and test("^[a-z0-9]+$"))
      and (.goarch | type == "string" and test("^[a-z0-9]+$"))
  ] | all)
  and ([.targets[] | "\(.goos)/\(.goarch)"] | length == (unique | length))
' "$targets_file" >/dev/null ||
  fail "release-targets.json is invalid"

project_name=$(jq -r '.project_name' "$targets_file")
module_path=$(jq -r '.module_path' "$targets_file")
checksum_name="${project_name}_${release_version}_SHA256SUMS"
checksum_path="$dist_dir/$checksum_name"
[[ -f "$checksum_path" ]] || fail "missing checksum file $checksum_name"

temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-artifacts.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT
expected_archives="$temporary_dir/expected-archives.txt"
actual_archives="$temporary_dir/actual-archives.txt"
expected_checksum_assets="$temporary_dir/expected-checksum-assets.txt"
actual_checksum_assets="$temporary_dir/actual-checksum-assets.txt"

jq -r --arg version "$release_version" '
  . as $config
  | .targets[]
  | "\($config.project_name)_\($version)_\(.goos)_\(.goarch).zip"
' "$targets_file" | LC_ALL=C sort >"$expected_archives"

find "$dist_dir" -maxdepth 1 -type f \
  -name "${project_name}_${release_version}_*.zip" \
  -exec basename {} \; | LC_ALL=C sort >"$actual_archives"

if ! diff -u "$expected_archives" "$actual_archives"; then
  fail "archive set does not exactly match release-targets.json"
fi

manifest_name="${project_name}_${release_version}_manifest.json"
manifest_path="$dist_dir/$manifest_name"
[[ -f "$manifest_path" ]] || fail "missing Registry manifest $manifest_name"
cmp "$repository_dir/terraform-registry-manifest.json" "$manifest_path" >/dev/null ||
  fail "$manifest_name does not match terraform-registry-manifest.json"

{
  cat "$expected_archives"
  printf '%s\n' "$manifest_name"
} | LC_ALL=C sort >"$expected_checksum_assets"

if ! awk '
  NF != 2 || length($1) != 64 || $1 !~ /^[0-9a-f]+$/ {
    invalid = 1
    next
  }
  {
    print $2
  }
  END {
    exit invalid
  }
' "$checksum_path" | LC_ALL=C sort >"$actual_checksum_assets"; then
  fail "$checksum_name contains malformed entries"
fi

if ! diff -u "$expected_checksum_assets" "$actual_checksum_assets"; then
  fail "$checksum_name must contain exactly the provider archives and Registry manifest"
fi

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

while IFS= read -r asset_name; do
  asset_path="$dist_dir/$asset_name"
  [[ -f "$asset_path" ]] || fail "missing checksummed asset $asset_name"
  expected_checksum=$(awk -v name="$asset_name" '$2 == name {print $1}' "$checksum_path")
  actual_checksum=$(hash_file "$asset_path")
  [[ "$actual_checksum" == "$expected_checksum" ]] ||
    fail "checksum mismatch for $asset_name"
done <"$expected_checksum_assets"

while IFS=$'\t' read -r goos goarch archive_name; do
  archive_path="$dist_dir/$archive_name"
  binary_name="${project_name}_v${release_version}"
  if [[ "$goos" == "windows" ]]; then
    binary_name="${binary_name}.exe"
  fi

  binary_count=$(unzip -Z1 "$archive_path" | awk -v name="$binary_name" '$0 == name {count++} END {print count + 0}')
  [[ "$binary_count" == "1" ]] ||
    fail "$archive_name contains $binary_count copies of $binary_name, want exactly 1"

  extract_dir="$temporary_dir/${goos}_${goarch}"
  mkdir -p "$extract_dir"
  unzip -qq "$archive_path" "$binary_name" -d "$extract_dir"
  binary_path="$extract_dir/$binary_name"
  [[ -f "$binary_path" ]] || fail "failed to extract $binary_name from $archive_name"

  metadata=$(go version -m "$binary_path") ||
    fail "cannot read Go build metadata from $archive_name"
  embedded_revision=$(awk '$1 == "build" && $2 ~ /^vcs.revision=/ {
    sub(/^vcs.revision=/, "", $2)
    print $2
  }' <<<"$metadata")
  embedded_modified=$(awk '$1 == "build" && $2 ~ /^vcs.modified=/ {
    sub(/^vcs.modified=/, "", $2)
    print $2
  }' <<<"$metadata")
  embedded_module_version=$(awk -v module="$module_path" '
    $1 == "mod" && $2 == module {print $3}
  ' <<<"$metadata")

  [[ "$embedded_revision" == "$expected_sha" ]] ||
    fail "$archive_name embeds revision ${embedded_revision:-missing}, want $expected_sha"
  [[ "$embedded_modified" == "false" ]] ||
    fail "$archive_name embeds vcs.modified=${embedded_modified:-missing}, want false"
  [[ "$embedded_module_version" == "$expected_module_version" ]] ||
    fail "$archive_name embeds module version ${embedded_module_version:-missing}, want $expected_module_version"
  [[ "$embedded_module_version" != *"+dirty"* ]] ||
    fail "$archive_name embeds a dirty module version"
done < <(
  jq -r --arg version "$release_version" '
    . as $config
    | .targets[]
    | [
        .goos,
        .goarch,
        "\($config.project_name)_\($version)_\(.goos)_\(.goarch).zip"
      ]
    | @tsv
  ' "$targets_file"
)

echo "release artifacts: verified $(wc -l <"$expected_archives" | tr -d ' ') archives at $expected_sha"
