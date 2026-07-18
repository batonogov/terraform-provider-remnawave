resource "remnawave_hwid_device" "phone" {
  user_uuid = remnawave_user.example.uuid
  hwid      = "device-fingerprint-abc123"
}
