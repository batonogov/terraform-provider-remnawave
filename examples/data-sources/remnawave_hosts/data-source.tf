data "remnawave_hosts" "all" {}

output "all_hosts" {
  value = data.remnawave_hosts.all.hosts
}
