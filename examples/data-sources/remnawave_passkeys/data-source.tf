data "remnawave_passkeys" "current" {}

output "passkey_count" {
  value = length(data.remnawave_passkeys.current.passkeys)
}
