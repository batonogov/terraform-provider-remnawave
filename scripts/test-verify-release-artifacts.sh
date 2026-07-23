#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/release-artifacts-test.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT
export GOMODCACHE="$temporary_dir/gomodcache"
fixture_repo="$temporary_dir/repository"
dist_dir="$temporary_dir/dist"
build_dir="$temporary_dir/build"
project_name=$(jq -r '.project_name' "$repository_dir/release-targets.json")
module_path=$(jq -r '.module_path' "$repository_dir/release-targets.json")
release_version=1.2.3

mkdir -p "$fixture_repo" "$dist_dir" "$build_dir"
(
  cd "$fixture_repo"
  git init -q
  git config user.name "Release Test"
  git config user.email "release-test@example.com"
  printf 'module %s\n\ngo 1.26.5\n' "$module_path" >go.mod
  printf 'package main\n\nimport "fmt"\n\nvar version = "dev"\n\nfunc main() { fmt.Println(version) }\n' >main.go
  git add go.mod main.go
  git -c commit.gpgsign=false commit -qm "test fixture"
  git tag "v$release_version"
  go build -trimpath -ldflags="-X main.version=$release_version" -o "$build_dir/$project_name" .
)

fixture_sha=$(git -C "$fixture_repo" rev-parse HEAD)
fixture_module_version=$(go version -m "$build_dir/$project_name" | awk -v module="$module_path" '
  $1 == "mod" && $2 == module {print $3}
')
[[ -n "$fixture_module_version" ]] || {
  echo "fixture binary is missing module version metadata" >&2
  exit 1
}

create_archives() {
  local binary_path=$1
  while IFS=$'\t' read -r goos goarch; do
    archive_name="${project_name}_${release_version}_${goos}_${goarch}.zip"
    binary_name="${project_name}_v${release_version}"
    if [[ "$goos" == "windows" ]]; then
      binary_name="${binary_name}.exe"
    fi
    cp "$binary_path" "$build_dir/$binary_name"
    (
      cd "$build_dir"
      zip -q -j "$dist_dir/$archive_name" "$binary_name"
    )
    rm "$build_dir/$binary_name"
  done < <(jq -r '.targets[] | [.goos, .goarch] | @tsv' "$repository_dir/release-targets.json")
}

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

write_checksums() {
  (
    cd "$dist_dir"
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum ./*.zip "./${project_name}_${release_version}_manifest.json"
    else
      shasum -a 256 ./*.zip "./${project_name}_${release_version}_manifest.json"
    fi
  ) | sed 's#  \./#  #' >"$dist_dir/${project_name}_${release_version}_SHA256SUMS"
}

run_check() {
  "$script_dir/verify-release-artifacts.sh" \
    "$dist_dir" \
    "$release_version" \
    "${EXPECTED_SHA:-$fixture_sha}" \
    "${EXPECTED_MODULE_VERSION:-$fixture_module_version}"
}

expect_failure() {
  local name=$1
  if run_check >/dev/null 2>&1; then
    echo "$name: artifact verification unexpectedly succeeded" >&2
    exit 1
  fi
}

create_archives "$build_dir/$project_name"
cp \
  "$repository_dir/terraform-registry-manifest.json" \
  "$dist_dir/${project_name}_${release_version}_manifest.json"
write_checksums
run_check >/dev/null

unexpected_sbom="${project_name}_${release_version}_linux_amd64.zip.spdx.json"
printf '{}\n' >"$dist_dir/$unexpected_sbom"
printf '%s  %s\n' \
  "$(hash_file "$dist_dir/$unexpected_sbom")" \
  "$unexpected_sbom" \
  >>"$dist_dir/${project_name}_${release_version}_SHA256SUMS"
expect_failure "SBOM checksum subject"
rm "$dist_dir/$unexpected_sbom"
write_checksums

missing_archive=$(head -n 1 < <(
  jq -r --arg version "$release_version" '
    . as $config
    | .targets[]
    | "\($config.project_name)_\($version)_\(.goos)_\(.goarch).zip"
  ' "$repository_dir/release-targets.json"
))
mv "$dist_dir/$missing_archive" "$temporary_dir/$missing_archive"
expect_failure "missing archive"
mv "$temporary_dir/$missing_archive" "$dist_dir/$missing_archive"

EXPECTED_SHA=0000000000000000000000000000000000000000 expect_failure "wrong embedded revision"
EXPECTED_MODULE_VERSION=v9.9.9 expect_failure "wrong module version"

tampered_archive="${project_name}_${release_version}_darwin_amd64.zip"
cp "$dist_dir/$tampered_archive" "$temporary_dir/$tampered_archive"
printf 'tampered\n' >>"$dist_dir/$tampered_archive"
expect_failure "tampered archive"
mv "$temporary_dir/$tampered_archive" "$dist_dir/$tampered_archive"

printf '\n' >>"$fixture_repo/main.go"
(
  cd "$fixture_repo"
  go build -trimpath -ldflags="-X main.version=$release_version" -o "$build_dir/dirty-provider" .
)
dirty_archive="${project_name}_${release_version}_linux_amd64.zip"
cp "$build_dir/dirty-provider" "$build_dir/${project_name}_v${release_version}"
(
  cd "$build_dir"
  zip -q -j -FS "$dist_dir/$dirty_archive" "${project_name}_v${release_version}"
)
rm "$build_dir/${project_name}_v${release_version}"
write_checksums
expect_failure "dirty build metadata"

echo "release artifact tests passed"
