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

echo "example validation: validating generated resource and data-source examples"
fixture_dir="$script_dir/testdata/example-fixtures"
validated_count=0

while IFS= read -r example_dir; do
  relative_dir=${example_dir#"$repository_dir/examples/"}
  module_name=$(printf '%s' "$relative_dir" | tr '/' '-')
  module_dir="$temporary_dir/module-$module_name"
  data_dir="$temporary_dir/terraform-data-$module_name"
  mkdir -p "$module_dir" "$data_dir"

  cp "$repository_dir/examples/provider/provider.tf" "$module_dir/provider.tf"
  cp "$example_dir"/*.tf "$module_dir/"

  case "$relative_dir" in
    resources/remnawave_billing_history)
      cp "$fixture_dir/infra-provider-example.tf" "$module_dir/fixture-infra-provider.tf"
      ;;
    resources/remnawave_drop_connections|resources/remnawave_hwid_device|resources/remnawave_user_metadata)
      cp "$fixture_dir/user-example.tf" "$module_dir/fixture-user.tf"
      ;;
    resources/remnawave_host|resources/remnawave_internal_squad|resources/remnawave_node)
      cp "$fixture_dir/config-profile-default.tf" "$module_dir/fixture-config-profile.tf"
      ;;
    resources/remnawave_node_metadata)
      cp "$fixture_dir/config-profile-default.tf" "$module_dir/fixture-config-profile.tf"
      cp "$fixture_dir/node-de-fra-01.tf" "$module_dir/fixture-node.tf"
      ;;
  esac

  validation_output="$temporary_dir/validate-$module_name.log"
  if ! TF_CLI_CONFIG_FILE="$cli_config" \
    TF_DATA_DIR="$data_dir" \
    terraform -chdir="$module_dir" validate -no-color >"$validation_output" 2>&1; then
    echo "example validation: failed for examples/$relative_dir" >&2
    cat "$validation_output" >&2
    exit 1
  fi
  validated_count=$((validated_count + 1))
done < <(
  find "$repository_dir/examples/resources" "$repository_dir/examples/data-sources" \
    -mindepth 1 -maxdepth 1 -type d | sort
)

echo "example validation: validated $validated_count generated examples"
