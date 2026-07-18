resource "remnawave_host" "vless" {
  remark                      = "🇩🇪 Frankfurt"
  address                     = "vpn.example.com"
  port                        = 443
  config_profile_uuid         = remnawave_config_profile.default.uuid
  config_profile_inbound_uuid = remnawave_config_profile.default.inbounds[0].uuid
  tags                        = ["EU", "PREMIUM"]
}
