#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "release supply chain: $*" >&2
  exit 1
}

[[ $# -eq 3 ]] ||
  fail "usage: verify-release-supply-chain.sh DIST_DIR VERSION ARCHIVE_CHECKSUMS_OUTPUT"

dist_dir=$1
release_version=$2
archive_checksums_output=$3
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
targets_file="$repository_dir/release-targets.json"

[[ -d "$dist_dir" ]] || fail "distribution directory does not exist: $dist_dir"
[[ "$release_version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] ||
  fail "release version is not strict semantic versioning"
[[ -d "$(dirname "$archive_checksums_output")" ]] ||
  fail "output directory does not exist"

project_name=$(jq -r '.project_name' "$targets_file")
checksum_name="${project_name}_${release_version}_SHA256SUMS"
checksum_path="$dist_dir/$checksum_name"
[[ -f "$checksum_path" ]] || fail "missing checksum file $checksum_name"

temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-supply-chain.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT
expected_sboms="$temporary_dir/expected-sboms.txt"
actual_sboms="$temporary_dir/actual-sboms.txt"
archive_checksums="$temporary_dir/archive-SHA256SUMS"

jq -r --arg version "$release_version" '
  . as $config
  | .targets[]
  | "\($config.project_name)_\($version)_\(.goos)_\(.goarch).zip.spdx.json"
' "$targets_file" | LC_ALL=C sort >"$expected_sboms"

find "$dist_dir" -maxdepth 1 -type f -name '*.zip.spdx.json' \
  -exec basename {} \; | LC_ALL=C sort >"$actual_sboms"
if ! diff -u "$expected_sboms" "$actual_sboms"; then
  fail "SBOM set does not exactly match the release archive set"
fi

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

: >"$archive_checksums"
while IFS= read -r sbom_name; do
  archive_name=${sbom_name%.spdx.json}
  archive_path="$dist_dir/$archive_name"
  sbom_path="$dist_dir/$sbom_name"
  [[ -f "$archive_path" ]] || fail "missing archive $archive_name for $sbom_name"

  for asset_name in "$archive_name" "$sbom_name"; do
    checksum_count=$(awk -v name="$asset_name" '$2 == name {count++} END {print count + 0}' "$checksum_path")
    [[ "$checksum_count" == "1" ]] ||
      fail "$asset_name has $checksum_count checksum entries, want exactly 1"
    expected_checksum=$(awk -v name="$asset_name" '$2 == name {print $1}' "$checksum_path")
    actual_checksum=$(hash_file "$dist_dir/$asset_name")
    [[ "$actual_checksum" == "$expected_checksum" ]] ||
      fail "checksum mismatch for $asset_name"
  done

  jq -e --arg archive "$archive_name" '
    def nonempty: type == "string" and length > 0;
    (.spdxVersion == "SPDX-2.3")
    and (.dataLicense == "CC0-1.0")
    and (.SPDXID == "SPDXRef-DOCUMENT")
    and (.name | nonempty and endswith($archive))
    and (.documentNamespace | nonempty)
    and (.creationInfo.created | nonempty)
    and (.creationInfo.creators
      | type == "array" and any(.[]; type == "string" and test("syft"; "i")))
    and (.packages
      | type == "array"
        and length > 0
        and all(.[];
          (.SPDXID | nonempty)
          and (.name | nonempty)
          and (.versionInfo | nonempty)
          and (.licenseConcluded | nonempty)
          and (.licenseDeclared | nonempty)
        ))
    and (.relationships
      | type == "array"
        and length > 0
        and any(.[]; .relationshipType == "DESCRIBES"))
  ' "$sbom_path" >/dev/null ||
    fail "$sbom_name is not a complete Syft SPDX 2.3 document"

  archive_checksum=$(awk -v name="$archive_name" '$2 == name {print $1}' "$checksum_path")
  printf '%s  %s\n' "$archive_checksum" "$archive_name" >>"$archive_checksums"
done <"$expected_sboms"

expected_count=$(wc -l <"$expected_sboms" | tr -d ' ')
actual_count=$(wc -l <"$archive_checksums" | tr -d ' ')
[[ "$actual_count" == "$expected_count" ]] ||
  fail "archive attestation subject count is $actual_count, want $expected_count"

cp "$archive_checksums" "$archive_checksums_output"
echo "release supply chain: verified $expected_count archive/SBOM pairs"
