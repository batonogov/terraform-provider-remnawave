#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "release publication: $*" >&2
  exit 1
}

[[ $# -eq 5 ]] ||
  fail "usage: publish-release-draft.sh RELEASE_ID RELEASE_UPLOAD_URL TAG_NAME EXPECTED_SHA DIST_DIR"

release_id=$1
release_upload_url=$2
tag_name=$3
expected_sha=$4
dist_dir=$5
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
targets_file="$repository_dir/release-targets.json"

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
: "${GH_TOKEN:?GH_TOKEN is required}"

[[ "$release_id" =~ ^[1-9][0-9]*$ ]] ||
  fail "release ID must be a positive integer"
[[ "$GITHUB_REPOSITORY" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] ||
  fail "GITHUB_REPOSITORY is invalid"
[[ "$tag_name" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] ||
  fail "tag name is not strict semantic versioning"
[[ "$expected_sha" =~ ^[0-9a-f]{40}$ ]] ||
  fail "expected SHA must be a full lowercase commit SHA"
[[ -d "$dist_dir" ]] ||
  fail "distribution directory does not exist: $dist_dir"
[[ -f "$targets_file" ]] ||
  fail "release-targets.json is missing"

expected_upload_url="https://uploads.github.com/repos/${GITHUB_REPOSITORY}/releases/${release_id}/assets{?name,label}"
[[ "$release_upload_url" == "$expected_upload_url" ]] ||
  fail "release upload URL does not match repository and release ID"

release_version=${tag_name#v}
project_name=$(jq -er '.project_name | select(type == "string" and length > 0)' "$targets_file") ||
  fail "release-targets.json has no valid project name"
checksum_name="${project_name}_${release_version}_SHA256SUMS"
signature_name="${checksum_name}.sig"
manifest_name="${project_name}_${release_version}_manifest.json"
bundle_name="${project_name}_${release_version}_provenance.intoto.jsonl"
checksum_file="$dist_dir/$checksum_name"
signature_file="$dist_dir/$signature_name"
bundle_file="$dist_dir/$bundle_name"

for required_file in "$checksum_file" "$signature_file" "$bundle_file"; do
  [[ -f "$required_file" ]] ||
    fail "missing local release file $(basename "$required_file")"
done

temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-publication.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT
release_json="$temporary_dir/release.json"
published_json="$temporary_dir/published.json"
upload_json="$temporary_dir/upload.json"
tag_json="$temporary_dir/tag.json"
expected_base_assets="$temporary_dir/expected-base-assets.txt"
expected_final_assets="$temporary_dir/expected-final-assets.txt"
remote_assets="$temporary_dir/remote-assets.txt"
release_api="repos/$GITHUB_REPOSITORY/releases/$release_id"
tag_api="repos/$GITHUB_REPOSITORY/git/ref/tags/$tag_name"
upload_api="https://uploads.github.com/repos/$GITHUB_REPOSITORY/releases/$release_id/assets"

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

validate_release() {
  local json_file=$1
  local expected_draft=$2

  jq -e \
    --argjson release_id "$release_id" \
    --arg upload_url "$release_upload_url" \
    --arg tag_name "$tag_name" \
    --arg expected_sha "$expected_sha" \
    --argjson expected_draft "$expected_draft" '
      .id == $release_id
      and .upload_url == $upload_url
      and .tag_name == $tag_name
      and .target_commitish == $expected_sha
      and .draft == $expected_draft
      and .prerelease == false
      and (
        if $expected_draft
        then .published_at == null
        else (.published_at | type == "string" and length > 0)
        end
      )
      and (.assets | type == "array")
      and ([.assets[] | select(
        (.name | type != "string" or length == 0)
        or .state != "uploaded"
      )] | length == 0)
    ' "$json_file" >/dev/null ||
    fail "release identity or state does not match the verified draft"
}

write_remote_assets() {
  local json_file=$1
  jq -r '.assets[].name' "$json_file" | LC_ALL=C sort >"$remote_assets"
}

show_asset_diff() {
  local expected_file=$1
  diff -u "$expected_file" "$remote_assets" || :
}

verify_tag_target() {
  gh api "$tag_api" >"$tag_json"
  jq -e \
    --arg expected_ref "refs/tags/$tag_name" \
    --arg expected_sha "$expected_sha" '
      .ref == $expected_ref
      and .object.type == "commit"
      and .object.sha == $expected_sha
  ' "$tag_json" >/dev/null ||
    fail "release tag no longer resolves to the CI-verified commit"
}

verify_remote_asset_digests() {
  local json_file=$1
  local expected_assets_file=$2
  local asset_name
  local asset_path
  local expected_digest

  while IFS= read -r asset_name; do
    asset_path="$dist_dir/$asset_name"
    if [[ "$asset_name" == "$manifest_name" ]]; then
      asset_path="$repository_dir/terraform-registry-manifest.json"
    fi
    [[ -f "$asset_path" ]] ||
      fail "missing verified local asset $asset_name"
    expected_digest="sha256:$(hash_file "$asset_path")"
    jq -e \
      --arg asset_name "$asset_name" \
      --arg expected_digest "$expected_digest" '
        [.assets[] | select(.name == $asset_name)]
        | length == 1
          and .[0].state == "uploaded"
          and .[0].digest == $expected_digest
      ' "$json_file" >/dev/null ||
      fail "remote asset $asset_name does not match the verified local file"
  done <"$expected_assets_file"
}

{
  jq -r --arg version "$release_version" '
    . as $config
    | .targets[]
    | "\($config.project_name)_\($version)_\(.goos)_\(.goarch).zip"
  ' "$targets_file"
  jq -r --arg version "$release_version" '
    . as $config
    | .targets[]
    | "\($config.project_name)_\($version)_\(.goos)_\(.goarch).zip.spdx.json"
  ' "$targets_file"
  printf '%s\n' "$manifest_name" "$checksum_name" "$signature_name"
} | LC_ALL=C sort >"$expected_base_assets"

{
  cat "$expected_base_assets"
  printf '%s\n' "$bundle_name"
} | LC_ALL=C sort >"$expected_final_assets"

verify_tag_target
gh api "$release_api" >"$release_json"
write_remote_assets "$release_json"

if jq -e '.draft == false' "$release_json" >/dev/null; then
  validate_release "$release_json" false
  if ! cmp -s "$expected_final_assets" "$remote_assets"; then
    show_asset_diff "$expected_final_assets"
    fail "published release asset set does not match the verified set"
  fi
  verify_remote_asset_digests "$release_json" "$expected_final_assets"
  echo "release publication: $tag_name was already published from verified release ID $release_id"
  exit 0
fi

validate_release "$release_json" true
if cmp -s "$expected_base_assets" "$remote_assets"; then
  verify_remote_asset_digests "$release_json" "$expected_base_assets"
  gh api \
    --method POST \
    "$upload_api?name=$bundle_name" \
    --header "Content-Type: application/octet-stream" \
    --input "$bundle_file" >"$upload_json"
  jq -e --arg bundle_name "$bundle_name" '
    .name == $bundle_name and .state == "uploaded"
  ' "$upload_json" >/dev/null ||
    fail "GitHub did not confirm the provenance upload"
elif cmp -s "$expected_final_assets" "$remote_assets"; then
  verify_remote_asset_digests "$release_json" "$expected_final_assets"
else
  show_asset_diff "$expected_final_assets"
  fail "draft release asset set is not safe to publish"
fi

gh api "$release_api" >"$release_json"
validate_release "$release_json" true
write_remote_assets "$release_json"
if ! cmp -s "$expected_final_assets" "$remote_assets"; then
  show_asset_diff "$expected_final_assets"
  fail "draft release does not contain the exact final asset set"
fi
verify_remote_asset_digests "$release_json" "$expected_final_assets"
verify_tag_target

gh api --method PATCH "$release_api" -F draft=false >"$published_json"
validate_release "$published_json" false
write_remote_assets "$published_json"
if ! cmp -s "$expected_final_assets" "$remote_assets"; then
  show_asset_diff "$expected_final_assets"
  fail "published release asset set changed during publication"
fi
verify_remote_asset_digests "$published_json" "$expected_final_assets"

gh api "$release_api" >"$release_json"
validate_release "$release_json" false
write_remote_assets "$release_json"
if ! cmp -s "$expected_final_assets" "$remote_assets"; then
  show_asset_diff "$expected_final_assets"
  fail "published release asset set does not match the verified set"
fi
verify_remote_asset_digests "$release_json" "$expected_final_assets"
verify_tag_target

echo "release publication: published $tag_name from verified release ID $release_id"
