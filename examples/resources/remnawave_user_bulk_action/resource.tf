# Extend subscriptions for selected users by 30 days.
resource "remnawave_user_bulk_action" "extend_expiration" {
  action = "extend_expiration"
  uuids = [
    "550e8400-e29b-41d4-a716-446655440000",
    "550e8400-e29b-41d4-a716-446655440001",
  ]
  days = 30

  triggers = {
    change_ticket = "OPS-1234"
  }
}
