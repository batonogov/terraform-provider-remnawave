data "remnawave_nodes_metrics" "live" {}

output "node_metrics" {
  value = data.remnawave_nodes_metrics.live
}
