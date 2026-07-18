resource "remnawave_node_metadata" "info" {
  node_uuid = remnawave_node.de_fra_01.uuid
  metadata = jsonencode({
    datacenter = "fra1"
    provider   = "hetzner"
  })
}
