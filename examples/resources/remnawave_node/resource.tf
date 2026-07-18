resource "remnawave_node" "de_fra_01" {
  name                    = "de-fra-01"
  address                 = "1.2.3.4"
  port                    = 443
  config_profile_uuid     = remnawave_config_profile.default.uuid
  config_profile_inbounds = [remnawave_config_profile.default.inbounds[0].uuid]
}
