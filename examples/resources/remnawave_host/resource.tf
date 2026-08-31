resource "remnawave_host" "vless" {
  remark                      = "🇩🇪 Frankfurt"
  address                     = "vpn.example.com"
  port                        = 443
  config_profile_uuid         = remnawave_config_profile.default.uuid
  config_profile_inbound_uuid = remnawave_config_profile.default.inbounds[0].uuid
  tags                        = ["EU", "PREMIUM"]

  # Remnawave 3.3+: rewrite generated client configuration per output format.
  mapper = jsonencode({
    mihomo  = [{ op = "set", to = "tfo", value = true }]
    singbox = [{ op = "set", to = "tcp_fast_open", value = true }]
  })

  # Remnawave 3.4+: control internal squad visibility explicitly.
  # mode = "EXCLUDE" hides the host from the listed squads,
  # mode = "ALLOW_ONLY" shows it only in the listed squads.
  # internal_squads_mode = "EXCLUDE"
  # internal_squads      = [remnawave_internal_squad.free.uuid]
}
