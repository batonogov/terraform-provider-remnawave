# Restart selected nodes during a maintenance window.
resource "remnawave_node_bulk_action" "restart" {
  action = "restart"
  uuids = [
    "550e8400-e29b-41d4-a716-446655440000",
    "550e8400-e29b-41d4-a716-446655440001",
  ]

  triggers = {
    maintenance_window = "2026-01"
  }
}
