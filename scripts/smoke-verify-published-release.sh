#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/smoke-verify-published-release.sh vX.Y.Z

Downloads and verifies one published release. RELEASE_VERIFICATION_ASSET_DIR
may point to an existing asset directory for offline testing.
RELEASE_KEY_METADATA_FILE may point to saved Terraform Registry download
metadata instead of fetching it.
EOF
}

[[ $# == 1 ]] || {
  usage >&2
  exit 2
}
tag=$1
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]] || {
  echo "release verification: tag must be strict SemVer with a v prefix" >&2
  exit 2
}

for command in gh gpg jq; do
  command -v "$command" >/dev/null || {
    echo "release verification: $command is required" >&2
    exit 1
  }
done

repo=${RELEASE_VERIFICATION_REPOSITORY:-batonogov/terraform-provider-remnawave}
provider_address=${RELEASE_VERIFICATION_PROVIDER_ADDRESS:-batonogov/remnawave}
version=${tag#v}
expected_fingerprint=CB77A6037F6CA36D514C8DC5B6D212FC24D5A5B1
expected_key_id=${expected_fingerprint: -16}
project_name=terraform-provider-remnawave
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/published-release-verification.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

metadata_file=${RELEASE_KEY_METADATA_FILE:-}
if [[ -z "$metadata_file" ]]; then
  command -v curl >/dev/null || {
    echo "release verification: curl is required" >&2
    exit 1
  }
  metadata_file="$temporary_dir/registry-metadata.json"
  registry_url="https://registry.terraform.io/v1/providers/$provider_address/$version/download/linux/amd64"
  curl -fsSL "$registry_url" >"$metadata_file"
fi

key_file="$temporary_dir/release-key.asc"
jq -er --arg key_id "$expected_key_id" '
  [.signing_keys.gpg_public_keys[]
    | select((.key_id | ascii_upcase) == $key_id)
    | .ascii_armor]
  | if length == 1 then .[0] else error("expected exactly one release key") end
' "$metadata_file" >"$key_file"

gpg_home="$temporary_dir/gnupg"
mkdir -m 700 "$gpg_home"
actual_fingerprint=$(
  gpg --batch --homedir "$gpg_home" --show-keys --with-colons "$key_file" |
    awk -F: '$1 == "fpr" {print toupper($10); exit}'
)
[[ "$actual_fingerprint" == "$expected_fingerprint" ]] || {
  echo \
    "release verification: fingerprint mismatch: expected $expected_fingerprint, got ${actual_fingerprint:-none}" \
    >&2
  exit 1
}
gpg --batch --homedir "$gpg_home" --import "$key_file" >/dev/null

asset_dir=${RELEASE_VERIFICATION_ASSET_DIR:-}
if [[ -z "$asset_dir" ]]; then
  asset_dir="$temporary_dir/assets"
  mkdir "$asset_dir"
  gh release download "$tag" --repo "$repo" --dir "$asset_dir"
fi
[[ -d "$asset_dir" ]] || {
  echo "release verification: asset directory does not exist: $asset_dir" >&2
  exit 1
}

checksums="${project_name}_${version}_SHA256SUMS"
signature="${checksums}.sig"
manifest="${project_name}_${version}_manifest.json"
bundle="${project_name}_${version}_provenance.intoto.jsonl"
archive="${project_name}_${version}_linux_amd64.zip"
sbom="${archive}.spdx.json"
for required_asset in \
  "$checksums" \
  "$signature" \
  "$manifest" \
  "$bundle" \
  "$archive" \
  "$sbom"; do
  [[ -f "$asset_dir/$required_asset" ]] || {
    echo "release verification: missing asset $required_asset" >&2
    exit 1
  }
done

gpg --batch --homedir "$gpg_home" \
  --verify "$asset_dir/$signature" "$asset_dir/$checksums"

checksum_entries="$temporary_dir/checksum-entries"
awk '{print $2}' "$asset_dir/$checksums" >"$checksum_entries"
[[ $(grep -Ec '\.zip$' "$checksum_entries") -ge 1 ]] ||
  {
    echo "release verification: checksum manifest contains no provider archives" >&2
    exit 1
  }
[[ $(grep -Fxc "$manifest" "$checksum_entries") == 1 ]] ||
  {
    echo "release verification: checksum manifest must contain the Registry manifest exactly once" >&2
    exit 1
  }
if ! awk -v manifest="$manifest" '
  /\.zip$/ { next }
  $0 == manifest { next }
  { exit 1 }
' "$checksum_entries"; then
  echo "release verification: checksum manifest contains non-Registry assets" >&2
  exit 1
fi

(
  cd "$asset_dir"
  if command -v sha256sum >/dev/null; then
    sha256sum -c "$checksums"
  else
    shasum -a 256 -c "$checksums"
  fi
)

jq -e '
  .spdxVersion == "SPDX-2.3"
  and (.packages | type == "array" and length > 0)
  and (.relationships | type == "array" and length > 0)
' "$asset_dir/$sbom" >/dev/null

tag_sha=$(gh api "repos/$repo/commits/$tag" --jq .sha)
[[ "$tag_sha" =~ ^[0-9a-f]{40}$ ]] || {
  echo "release verification: could not resolve tag commit" >&2
  exit 1
}
for subject in "$archive" "$sbom"; do
  gh attestation verify "$asset_dir/$subject" \
    --bundle "$asset_dir/$bundle" \
    --repo "$repo" \
    --signer-workflow "$repo/.github/workflows/release-please.yml" \
    --source-digest "$tag_sha"
done

echo "release verification: $tag passed checksum, GPG, SBOM, and provenance checks"
