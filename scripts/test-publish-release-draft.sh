#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-publication-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT
mock_bin="$temporary_dir/bin"
dist_dir="$temporary_dir/dist"
state_file="$temporary_dir/release.json"
command_log="$temporary_dir/gh.log"
base_assets="$temporary_dir/base-assets.txt"
release_id=358689832
tag_name=v1.2.3
release_version=${tag_name#v}
expected_sha=1111111111111111111111111111111111111111
repository=batonogov/terraform-provider-remnawave
upload_url="https://uploads.github.com/repos/${repository}/releases/${release_id}/assets{?name,label}"
project_name=$(jq -r '.project_name' "$repository_dir/release-targets.json")
checksum_name="${project_name}_${release_version}_SHA256SUMS"
signature_name="${checksum_name}.sig"
manifest_name="${project_name}_${release_version}_manifest.json"
bundle_name="${project_name}_${release_version}_provenance.intoto.jsonl"

mkdir -p "$mock_bin" "$dist_dir"
printf 'checksums\n' >"$dist_dir/$checksum_name"
printf 'signature\n' >"$dist_dir/$signature_name"
printf 'provenance bundle\n' >"$dist_dir/$bundle_name"

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

{
  jq -r --arg version "$release_version" '
    . as $config
    | .targets[]
    | "\($config.project_name)_\($version)_\(.goos)_\(.goarch).zip"
  ' "$repository_dir/release-targets.json"
  jq -r --arg version "$release_version" '
    . as $config
    | .targets[]
    | "\($config.project_name)_\($version)_\(.goos)_\(.goarch).zip.spdx.json"
  ' "$repository_dir/release-targets.json"
  printf '%s\n' "$manifest_name" "$checksum_name" "$signature_name"
} | LC_ALL=C sort >"$base_assets"

while IFS= read -r asset_name; do
  if [[ "$asset_name" != "$manifest_name" ]]; then
    printf 'verified fixture for %s\n' "$asset_name" >"$dist_dir/$asset_name"
  fi
done <"$base_assets"

cat >"$mock_bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >>"$MOCK_GH_LOG"

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

[[ ${1:-} == api ]] || {
  echo "mock gh: only api is supported" >&2
  exit 1
}
shift

method=GET
endpoint=
input_file=
draft_field=
while [[ $# -gt 0 ]]; do
  case $1 in
    --method)
      method=$2
      shift 2
      ;;
    --header)
      shift 2
      ;;
    --input)
      input_file=$2
      shift 2
      ;;
    -F)
      draft_field=$2
      shift 2
      ;;
    *)
      [[ -z "$endpoint" ]] || {
        echo "mock gh: unexpected argument $1" >&2
        exit 1
      }
      endpoint=$1
      shift
      ;;
  esac
done

case $method in
  GET)
    if [[ "$endpoint" == "$MOCK_TAG_API" ]]; then
      jq -n \
        --arg ref "$MOCK_TAG_REF" \
        --arg sha "$MOCK_TAG_SHA" '{
          ref: $ref,
          object: {
            type: "commit",
            sha: $sha
          }
        }'
      exit 0
    fi
    [[ "$endpoint" == "$MOCK_RELEASE_API" ]] || {
      echo "mock gh: unexpected GET $endpoint" >&2
      exit 1
    }
    cat "$MOCK_RELEASE_STATE"
    ;;
  POST)
    [[ "$endpoint" == "$MOCK_UPLOAD_API?name=$MOCK_BUNDLE_NAME" ]] || {
      echo "mock gh: unexpected POST $endpoint" >&2
      exit 1
    }
    [[ -f "$input_file" ]] || {
      echo "mock gh: upload input is missing" >&2
      exit 1
    }
    digest="sha256:$(hash_file "$input_file")"
    jq \
      --arg name "$MOCK_BUNDLE_NAME" \
      --arg digest "$digest" '
        .assets += [{
          id: 999,
          name: $name,
          state: "uploaded",
          digest: $digest
        }]
      ' "$MOCK_RELEASE_STATE" >"${MOCK_RELEASE_STATE}.new"
    mv "${MOCK_RELEASE_STATE}.new" "$MOCK_RELEASE_STATE"
    jq --arg name "$MOCK_BUNDLE_NAME" '
      .assets[] | select(.name == $name)
    ' "$MOCK_RELEASE_STATE"
    ;;
  PATCH)
    [[ "$endpoint" == "$MOCK_RELEASE_API" ]] || {
      echo "mock gh: unexpected PATCH $endpoint" >&2
      exit 1
    }
    [[ "$draft_field" == "draft=false" ]] || {
      echo "mock gh: unexpected PATCH field $draft_field" >&2
      exit 1
    }
    if [[ ${MOCK_IGNORE_PATCH:-false} != true ]]; then
      jq '
        .draft = false
        | .published_at = "2026-01-01T00:00:00Z"
      ' "$MOCK_RELEASE_STATE" >"${MOCK_RELEASE_STATE}.new"
      mv "${MOCK_RELEASE_STATE}.new" "$MOCK_RELEASE_STATE"
    fi
    cat "$MOCK_RELEASE_STATE"
    ;;
  *)
    echo "mock gh: unsupported method $method" >&2
    exit 1
    ;;
esac
EOF
chmod +x "$mock_bin/gh"

export GH_TOKEN=test-token
export GITHUB_REPOSITORY=$repository
export MOCK_RELEASE_STATE=$state_file
export MOCK_GH_LOG=$command_log
export MOCK_RELEASE_API="repos/$repository/releases/$release_id"
export MOCK_TAG_API="repos/$repository/git/ref/tags/$tag_name"
export MOCK_TAG_REF="refs/tags/$tag_name"
export MOCK_TAG_SHA=$expected_sha
export MOCK_UPLOAD_API="https://uploads.github.com/repos/$repository/releases/$release_id/assets"
export MOCK_BUNDLE_NAME=$bundle_name

write_state() {
  local draft=${1:-true}
  local target_sha=${2:-$expected_sha}
  local include_bundle=${3:-false}
  local published_at=null
  local assets_json

  if [[ "$draft" == false ]]; then
    published_at='"2026-01-01T00:00:00Z"'
  fi

  assets_json=$(
    while IFS= read -r asset_name; do
      asset_path="$dist_dir/$asset_name"
      if [[ "$asset_name" == "$manifest_name" ]]; then
        asset_path="$repository_dir/terraform-registry-manifest.json"
      fi
      jq -n \
        --arg name "$asset_name" \
        --arg digest "sha256:$(hash_file "$asset_path")" '{
          id: 1,
          name: $name,
          state: "uploaded",
          digest: $digest
        }'
    done <"$base_assets" | jq -s .
  )

  if [[ "$include_bundle" == true ]]; then
    bundle_digest="sha256:$(hash_file "$dist_dir/$bundle_name")"
    assets_json=$(jq \
      --arg name "$bundle_name" \
      --arg digest "$bundle_digest" \
      '. + [{id: 999, name: $name, state: "uploaded", digest: $digest}]' \
      <<<"$assets_json")
  fi

  jq -n \
    --argjson id "$release_id" \
    --arg upload_url "$upload_url" \
    --arg tag_name "$tag_name" \
    --arg target_sha "$target_sha" \
    --argjson draft "$draft" \
    --argjson published_at "$published_at" \
    --argjson assets "$assets_json" '{
      id: $id,
      upload_url: $upload_url,
      tag_name: $tag_name,
      target_commitish: $target_sha,
      draft: $draft,
      prerelease: false,
      published_at: $published_at,
      assets: $assets
    }' >"$state_file"
  : >"$command_log"
}

run_publish() {
  PATH="$mock_bin:$PATH" \
    "$script_dir/publish-release-draft.sh" \
      "$release_id" \
      "$upload_url" \
      "$tag_name" \
      "$expected_sha" \
      "$dist_dir"
}

expect_failure_before_mutation() {
  local name=$1
  if run_publish >/dev/null 2>&1; then
    echo "$name: draft publication unexpectedly succeeded" >&2
    exit 1
  fi
  if grep -Eq -- '--method (POST|PATCH)' "$command_log"; then
    echo "$name: draft publication mutated GitHub before validation" >&2
    exit 1
  fi
}

write_state
run_publish >/dev/null
jq -e '
  .draft == false
  and .published_at != null
  and ([.assets[] | select(.name | endswith("_provenance.intoto.jsonl"))] | length == 1)
' "$state_file" >/dev/null
grep -Fq -- "--method POST $MOCK_UPLOAD_API?name=$bundle_name" "$command_log"
grep -Fq -- "--method PATCH $MOCK_RELEASE_API" "$command_log"
if grep -Fq '/releases/tags/' "$command_log"; then
  echo "successful publication used the endpoint that hides drafts" >&2
  exit 1
fi

write_state true "$expected_sha" true
run_publish >/dev/null
if grep -Fq -- '--method POST' "$command_log"; then
  echo "idempotent publication uploaded an existing matching bundle" >&2
  exit 1
fi

write_state false "$expected_sha" true
run_publish >/dev/null
if grep -Eq -- '--method (POST|PATCH)' "$command_log"; then
  echo "idempotent publication mutated an already-published verified release" >&2
  exit 1
fi

write_state true 2222222222222222222222222222222222222222
expect_failure_before_mutation "wrong release target"

write_state false
expect_failure_before_mutation "non-draft release"

write_state
jq '.assets[0].digest = "sha256:tampered"' "$state_file" >"${state_file}.new"
mv "${state_file}.new" "$state_file"
expect_failure_before_mutation "tampered release asset"

write_state
jq 'del(.assets[0])' "$state_file" >"${state_file}.new"
mv "${state_file}.new" "$state_file"
expect_failure_before_mutation "missing release asset"

write_state
export MOCK_TAG_SHA=2222222222222222222222222222222222222222
expect_failure_before_mutation "moved release tag"
export MOCK_TAG_SHA=$expected_sha

write_state
export MOCK_TAG_REF="refs/heads/$tag_name"
expect_failure_before_mutation "branch substituted for release tag"
export MOCK_TAG_REF="refs/tags/$tag_name"

write_state
if PATH="$mock_bin:$PATH" \
  "$script_dir/publish-release-draft.sh" \
    "$release_id" \
    "https://uploads.github.com/repos/attacker/repository/releases/$release_id/assets{?name,label}" \
    "$tag_name" \
    "$expected_sha" \
    "$dist_dir" >/dev/null 2>&1; then
  echo "wrong upload URL: draft publication unexpectedly succeeded" >&2
  exit 1
fi
[[ ! -s "$command_log" ]]

write_state
export MOCK_IGNORE_PATCH=true
if run_publish >/dev/null 2>&1; then
  echo "failed draft transition: publication unexpectedly succeeded" >&2
  exit 1
fi
unset MOCK_IGNORE_PATCH

echo "release draft publication tests passed"
