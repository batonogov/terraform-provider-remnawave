data "remnawave_keygen" "pubkey" {}

output "panel_public_key" {
  value     = data.remnawave_keygen.pubkey.pub_key
  sensitive = true
}
