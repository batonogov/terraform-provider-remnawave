resource "remnawave_user" "test" {
  username            = "my-user"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

# Reset traffic once on creation.
resource "remnawave_user_action" "reset" {
  user_uuid = remnawave_user.test.uuid
  action    = "reset_traffic"
  triggers  = [timestamp()]
}

# Extend expiration by 30 days (v3.0+ only).
resource "remnawave_user_action" "extend" {
  user_uuid = remnawave_user.test.uuid
  action    = "extend_expiration"
  days      = 30
  triggers  = [timestamp()]
}
