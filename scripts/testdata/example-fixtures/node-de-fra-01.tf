resource "remnawave_node" "de_fra_01" {
  name                    = "fixture-node"
  address                 = "192.0.2.1"
  config_profile_uuid     = remnawave_config_profile.default.uuid
  config_profile_inbounds = [remnawave_config_profile.default.inbounds[0].uuid]
}
