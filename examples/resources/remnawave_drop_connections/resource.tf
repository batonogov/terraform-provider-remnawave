resource "remnawave_drop_connections" "stale" {
  drop_by    = "user_uuids"
  user_uuids = [remnawave_user.example.uuid]
  triggers   = { timestamp = timestamp() }
}
