#!/usr/bin/env bash
set -euo pipefail

command -v go >/dev/null || {
  echo "example validation: go is required" >&2
  exit 1
}
command -v terraform >/dev/null || {
  echo "example validation: terraform is required" >&2
  exit 1
}

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/provider-example-validation.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT

plugin_dir="$temporary_dir/plugins"
mkdir -p "$plugin_dir"
(
  cd "$repository_dir"
  go build -o "$plugin_dir/terraform-provider-remnawave" .
)

cli_config="$temporary_dir/terraform.rc"
sed "s|__PLUGIN_DIR__|$plugin_dir|g" \
  "$script_dir/testdata/example-validation.tfrc.tmpl" >"$cli_config"

for example_dir in "$repository_dir/examples" "$repository_dir/examples/provider"; do
  example_name=$(basename "$example_dir")
  data_dir="$temporary_dir/terraform-data-$example_name"
  mkdir -p "$data_dir"
  echo "example validation: validating ${example_dir#"$repository_dir/"}"
  TF_CLI_CONFIG_FILE="$cli_config" \
    TF_DATA_DIR="$data_dir" \
    terraform -chdir="$example_dir" validate -no-color
done
