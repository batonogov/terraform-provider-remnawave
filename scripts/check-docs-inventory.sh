#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_dir=$(cd -- "$script_dir/.." && pwd)
cd "$repository_dir"

fail() {
  echo "documentation inventory: $*" >&2
  exit 1
}

count_provider_registrations() {
  local function_name=$1
  local constructor_suffix=$2
  awk -v function_name="$function_name" -v constructor_suffix="$constructor_suffix" '
    $0 ~ "func \\(p \\*RemnawaveProvider\\) " function_name { capture = 1; next }
    capture && /^}/ { print count + 0; exit }
    capture && $0 ~ "^[[:space:]]+New[A-Za-z0-9]+" constructor_suffix ",$" { count++ }
  ' provider/provider.go
}

count_files() {
  find "$1" -type f -name "$2" | wc -l | tr -d ' '
}

count_directories() {
  find "$1" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' '
}

require_text() {
  local file=$1
  local expected=$2
  grep -Fq -- "$expected" "$file" || fail "$file is missing: $expected"
}

verify_surface_files() {
  local kind=$1
  local registered_count=$2
  local docs_dir=$3
  local examples_dir=$4
  shift 4

  local source_files=()
  local source_file
  for source_file in "$@"; do
    [[ "$source_file" == *_test.go ]] || source_files+=("$source_file")
  done

  local discovered_count=0
  local type_name
  while IFS= read -r type_name; do
    [ -n "$type_name" ] || continue
    [ -f "$docs_dir/$type_name.md" ] || fail "missing $kind documentation for remnawave_$type_name"
    [ -d "$examples_dir/remnawave_$type_name" ] || fail "missing $kind example directory for remnawave_$type_name"
    require_text README.md "](docs/${docs_dir#docs/}/$type_name.md)"
    discovered_count=$((discovered_count + 1))
  done < <(sed -nE 's/.*resp\.TypeName = "remnawave_([^"]+)".*/\1/p' "${source_files[@]}" | sort -u)

  [ "$discovered_count" -eq "$registered_count" ] || fail "registered $registered_count ${kind}s, found $discovered_count type names in schemas"
}

resource_count=$(count_provider_registrations Resources Resource)
data_source_count=$(count_provider_registrations DataSources DataSource)
resource_docs_count=$(count_files docs/resources '*.md')
data_source_docs_count=$(count_files docs/data-sources '*.md')
resource_examples_count=$(count_directories examples/resources)
data_source_examples_count=$(count_directories examples/data-sources)
client_operation_count=$(awk '/^func \(c \*Client\) [A-Z]/ { count++ } END { print count + 0 }' provider/*.go)
acceptance_test_count=$(awk '/^func TestAcc/ { count++ } END { print count + 0 }' provider/*_test.go)

[ "$resource_docs_count" -eq "$resource_count" ] || fail "expected $resource_count resource docs, found $resource_docs_count"
[ "$data_source_docs_count" -eq "$data_source_count" ] || fail "expected $data_source_count data-source docs, found $data_source_docs_count"
[ "$resource_examples_count" -eq "$resource_count" ] || fail "expected $resource_count resource example directories, found $resource_examples_count"
[ "$data_source_examples_count" -eq "$data_source_count" ] || fail "expected $data_source_count data-source example directories, found $data_source_examples_count"

verify_surface_files resource "$resource_count" docs/resources examples/resources provider/resource_*.go
verify_surface_files data-source "$data_source_count" docs/data-sources examples/data-sources provider/data_source*.go

readme_resource_links=$(awk '/\]\(docs\/resources\// { count++ } END { print count + 0 }' README.md)
readme_data_source_links=$(awk '/\]\(docs\/data-sources\// { count++ } END { print count + 0 }' README.md)
[ "$readme_resource_links" -eq "$resource_count" ] || fail "README lists $readme_resource_links of $resource_count resources"
[ "$readme_data_source_links" -eq "$data_source_count" ] || fail "README lists $readme_data_source_links of $data_source_count data sources"

require_text README.md "**$data_source_count data sources**"
require_text AGENTS.md "### Data Sources ($data_source_count)"
require_text API_COVERAGE.md "- Resources: $resource_count"
require_text API_COVERAGE.md "- Data sources: $data_source_count"
require_text API_COVERAGE.md "- Exported client operations: $client_operation_count"
require_text API_COVERAGE.md "- Acceptance test entry points: $acceptance_test_count"

provider_version=$(awk -F'"' '/"\."/ { print $4; exit }' .release-please-manifest.json)
provider_minor=$(printf '%s\n' "$provider_version" | awk -F. '{ print $1 "." $2 }')
provider_constraint="~> $provider_minor.0"
for file in README.md docs/index.md examples/getting-started/main.tf examples/provider.tf examples/provider/provider.tf; do
  require_text "$file" "version = \"$provider_constraint\""
  constraint_count=$(grep -Fc -- "version = \"$provider_constraint\"" "$file" || true)
  managed_constraint_count=$(grep -Fc -- "version = \"$provider_constraint\" # x-release-please-version" "$file" || true)
  [ "$managed_constraint_count" -eq "$constraint_count" ] || fail "$file must mark every provider version constraint with x-release-please-version"
  require_text release-please-config.json "\"$file\""
done

supported_import_count=0
for resource_file in provider/resource_*.go; do
  [[ "$resource_file" == *_test.go ]] && continue
  grep -Eq 'func \([^)]*\*[^)]*Resource\) ImportState\(' "$resource_file" || continue

  if grep -q 'Import not supported' "$resource_file"; then
    continue
  fi
  resource_name=$(basename "$resource_file" .go)
  resource_name=${resource_name#resource_}
  import_file="examples/resources/remnawave_$resource_name/import.sh"
  [ -f "$import_file" ] || fail "missing import example for remnawave_$resource_name"
  require_text "$import_file" "terraform import remnawave_$resource_name."
  supported_import_count=$((supported_import_count + 1))
done

documented_import_count=$(find examples/resources -type f -name import.sh | wc -l | tr -d ' ')
[ "$documented_import_count" -eq "$supported_import_count" ] || fail "expected $supported_import_count import examples, found $documented_import_count"

generated_import_count=$(awk '/^## Import$/ { count++ } END { print count + 0 }' docs/resources/*.md)
[ "$generated_import_count" -eq "$supported_import_count" ] || fail "expected $supported_import_count generated import sections, found $generated_import_count"

echo "documentation inventory: $resource_count resources, $data_source_count data sources, $supported_import_count imports, $client_operation_count client operations, $acceptance_test_count acceptance tests"
