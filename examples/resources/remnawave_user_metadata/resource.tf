resource "remnawave_user_metadata" "info" {
  user_uuid = remnawave_user.example.uuid
  metadata = jsonencode({
    plan  = "premium"
    notes = "VIP customer"
  })
}
