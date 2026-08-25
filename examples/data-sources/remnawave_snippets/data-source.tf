data "remnawave_snippets" "all" {}

output "snippet_names" {
  value = [for snippet in data.remnawave_snippets.all.snippets : snippet.name]
}
