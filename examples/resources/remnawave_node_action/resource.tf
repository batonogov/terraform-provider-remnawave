resource "remnawave_node_action" "restart" {
  node_uuid     = "550e8400-e29b-41d4-a716-446655440000"
  action        = "restart"
  force_restart = true
  triggers      = ["maintenance-window-2026-01"]
}

resource "remnawave_node_action" "reset_traffic" {
  node_uuid = "550e8400-e29b-41d4-a716-446655440000"
  action    = "reset_traffic"
  triggers  = ["billing-period-2026-01"]
}
