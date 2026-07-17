resource "remnawave_node_action" "restart" {
  node_uuid     = remnawave_node.example.uuid
  action        = "restart"
  force_restart = true
  triggers      = [timestamp()]
}

resource "remnawave_node_action" "reset_traffic" {
  node_uuid = remnawave_node.example.uuid
  action    = "reset-traffic"
  triggers  = [var.traffic_reset_trigger]
}
