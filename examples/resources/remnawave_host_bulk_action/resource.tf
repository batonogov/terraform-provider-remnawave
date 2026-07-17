# Bulk-disable selected hosts
resource "remnawave_host_bulk_action" "disable_maintenance" {
  action = "disable"
  uuids  = ["550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001"]
  triggers = {
    reason = "scheduled-maintenance-2024-01-15"
  }
}
