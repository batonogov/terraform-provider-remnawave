data "remnawave_system_nodes_stats" "current" {}

output "node_stats" {
  value = data.remnawave_system_nodes_stats.current
}
