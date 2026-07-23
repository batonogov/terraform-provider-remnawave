#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-supply-chain-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT
dist_dir="$temporary_dir/dist"
archive_checksums="$temporary_dir/archive-SHA256SUMS"
project_name=$(jq -r '.project_name' "$repository_dir/release-targets.json")
release_version=1.2.3
mkdir -p "$dist_dir"

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

while IFS=$'\t' read -r goos goarch; do
  archive_name="${project_name}_${release_version}_${goos}_${goarch}.zip"
  sbom_name="${archive_name}.spdx.json"
  printf 'archive fixture for %s/%s\n' "$goos" "$goarch" >"$dist_dir/$archive_name"
  jq -n --arg archive "$archive_name" --arg target "$goos/$goarch" '{
    spdxVersion: "SPDX-2.3",
    dataLicense: "CC0-1.0",
    SPDXID: "SPDXRef-DOCUMENT",
    name: $archive,
    documentNamespace: ("https://example.invalid/sbom/" + $target),
    creationInfo: {
      created: "2026-01-01T00:00:00Z",
      creators: ["Tool: syft-1.49.0"]
    },
    documentDescribes: ["SPDXRef-Package-provider"],
    packages: [{
      SPDXID: "SPDXRef-Package-provider",
      name: "github.com/batonogov/terraform-provider-remnawave",
      versionInfo: "v1.2.3",
      licenseConcluded: "MIT",
      licenseDeclared: "MIT"
    }],
    relationships: [{
      spdxElementId: "SPDXRef-DOCUMENT",
      relationshipType: "DESCRIBES",
      relatedSpdxElement: "SPDXRef-Package-provider"
    }]
  }' >"$dist_dir/$sbom_name"
done < <(jq -r '.targets[] | [.goos, .goarch] | @tsv' "$repository_dir/release-targets.json")

write_checksums() {
  checksum_file="$dist_dir/${project_name}_${release_version}_SHA256SUMS"
  : >"$checksum_file"
  for asset in "$dist_dir"/*.zip "$dist_dir"/*.zip.spdx.json; do
    printf '%s  %s\n' "$(hash_file "$asset")" "$(basename "$asset")" >>"$checksum_file"
  done
}

run_check() {
  "$script_dir/verify-release-supply-chain.sh" \
    "$dist_dir" "$release_version" "$archive_checksums"
}

expect_failure() {
  local name=$1
  if run_check >/dev/null 2>&1; then
    echo "$name: supply-chain verification unexpectedly succeeded" >&2
    exit 1
  fi
}

write_checksums
run_check >/dev/null
[[ "$(wc -l <"$archive_checksums" | tr -d ' ')" == "10" ]]

test_sbom="${project_name}_${release_version}_linux_amd64.zip.spdx.json"
mv "$dist_dir/$test_sbom" "$temporary_dir/$test_sbom"
expect_failure "missing SBOM"
mv "$temporary_dir/$test_sbom" "$dist_dir/$test_sbom"

cp "$dist_dir/$test_sbom" "$temporary_dir/valid-sbom.json"
printf 'tampered\n' >>"$dist_dir/$test_sbom"
expect_failure "tampered SBOM"
cp "$temporary_dir/valid-sbom.json" "$dist_dir/$test_sbom"

jq 'del(.relationships)' "$temporary_dir/valid-sbom.json" >"$dist_dir/$test_sbom"
write_checksums
expect_failure "incomplete SPDX"
cp "$temporary_dir/valid-sbom.json" "$dist_dir/$test_sbom"

extra_sbom="$dist_dir/unexpected.zip.spdx.json"
cp "$temporary_dir/valid-sbom.json" "$extra_sbom"
write_checksums
expect_failure "unexpected SBOM"
rm "$extra_sbom"

write_checksums
checksum_file="$dist_dir/${project_name}_${release_version}_SHA256SUMS"
head -n 1 "$checksum_file" >>"$checksum_file"
expect_failure "duplicate checksum subject"

jq -e '
  .draft == true
  and .["force-tag-creation"] == true
  and .["release-type"] == "simple"
  and (.packages["."] | type == "object")
' "$repository_dir/release-please-config.json" >/dev/null

ruby -ryaml -e '
  config = YAML.load_file(ARGV.fetch(0))
  sbom = config.fetch("sboms").fetch(0)
  release = config.fetch("release")
  abort "SBOM generation is not enabled for archives" unless
    sbom["artifacts"] == "archive" &&
    sbom["documents"] == ["${artifact}.spdx.json"] &&
    sbom["disable"] == false
  abort "release must remain a reusable draft" unless
    release["draft"] == true && release["use_existing_draft"] == true
' "$repository_dir/.goreleaser.yml"

grep -Fq 'anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610' \
  "$repository_dir/.github/workflows/release-please.yml"
grep -Fq 'actions/attest@f7c74d28b9d84cb8768d0b8ca14a4bac6ef463e6' \
  "$repository_dir/.github/workflows/release-please.yml"
grep -Fq 'id-token: write' "$repository_dir/.github/workflows/release-please.yml"
grep -Fq 'attestations: write' "$repository_dir/.github/workflows/release-please.yml"
grep -Fq 'gh release edit "$TAG_NAME" --draft=false' \
  "$repository_dir/.github/workflows/release-please.yml"

echo "release supply-chain tests passed"
