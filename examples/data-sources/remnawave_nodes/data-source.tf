data "remnawave_nodes" "all" {}

output "all_nodes" {
  value = data.remnawave_nodes.all.nodes
}
