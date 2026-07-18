resource "remnawave_internal_squad" "premium" {
  name     = "Premium"
  inbounds = [remnawave_config_profile.default.inbounds[0].uuid]
}
