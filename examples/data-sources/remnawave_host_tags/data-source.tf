data "remnawave_host_tags" "all" {}

output "all_host_tags" {
  value = data.remnawave_host_tags.all.tags
}
