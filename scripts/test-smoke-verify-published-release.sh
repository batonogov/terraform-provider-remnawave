#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-verification-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

mkdir -p "$temporary_dir/bin" "$temporary_dir/assets"
cp "$script_dir/testdata/mock-gh-release-verification" "$temporary_dir/bin/gh"
cp "$script_dir/testdata/mock-gpg-release-verification" "$temporary_dir/bin/gpg"
chmod +x "$temporary_dir/bin/gh" "$temporary_dir/bin/gpg"

tag=v9.8.7-fixture
version=${tag#v}
project_name=terraform-provider-remnawave
archive="${project_name}_${version}_linux_amd64.zip"
manifest="${project_name}_${version}_manifest.json"
checksums="${project_name}_${version}_SHA256SUMS"
signature="${checksums}.sig"
bundle="${project_name}_${version}_provenance.intoto.jsonl"
sbom="${archive}.spdx.json"

printf '%s\n' "fixture provider archive" >"$temporary_dir/assets/$archive"
printf '%s\n' '{"version":1}' >"$temporary_dir/assets/$manifest"
printf '%s\n' \
  '{"spdxVersion":"SPDX-2.3","packages":[{"name":"fixture"}],"relationships":[{"relationshipType":"CONTAINS"}]}' \
  >"$temporary_dir/assets/$sbom"
printf '%s\n' '{}' >"$temporary_dir/assets/$bundle"
: >"$temporary_dir/assets/$signature"
(
  cd "$temporary_dir/assets"
  if command -v sha256sum >/dev/null; then
    sha256sum "$archive" "$manifest" >"$checksums"
  else
    shasum -a 256 "$archive" "$manifest" >"$checksums"
  fi
)

jq -n \
  --arg key_id B6D212FC24D5A5B1 \
  --arg ascii_armor "fixture public key" \
  '{signing_keys:{gpg_public_keys:[{key_id:$key_id,ascii_armor:$ascii_armor}]}}' \
  >"$temporary_dir/registry-metadata.json"

run_smoke() {
  PATH="$temporary_dir/bin:$PATH" \
    GH_MOCK_LOG="$temporary_dir/gh.log" \
    RELEASE_VERIFICATION_ASSET_DIR="$temporary_dir/assets" \
    RELEASE_KEY_METADATA_FILE="$temporary_dir/registry-metadata.json" \
    "$script_dir/smoke-verify-published-release.sh" "$tag"
}

run_smoke >/dev/null
(( $(wc -l <"$temporary_dir/gh.log") == 2 )) || {
  echo "release verification test: expected two attestation checks" >&2
  exit 1
}

if MOCK_GPG_FINGERPRINT=0000000000000000000000000000000000000000 \
  run_smoke >/dev/null 2>&1; then
  echo "release verification test: fingerprint mismatch was accepted" >&2
  exit 1
fi

cp "$temporary_dir/assets/$checksums" "$temporary_dir/original-checksums"
(
  cd "$temporary_dir/assets"
  if command -v sha256sum >/dev/null; then
    sha256sum "$sbom" >>"$checksums"
  else
    shasum -a 256 "$sbom" >>"$checksums"
  fi
)
if run_smoke >/dev/null 2>&1; then
  echo "release verification test: non-Registry checksum entry was accepted" >&2
  exit 1
fi
mv "$temporary_dir/original-checksums" "$temporary_dir/assets/$checksums"

printf '%s\n' "tampered" >>"$temporary_dir/assets/$archive"
if run_smoke >/dev/null 2>&1; then
  echo "release verification test: checksum mismatch was accepted" >&2
  exit 1
fi

echo "published release verification tests passed"
