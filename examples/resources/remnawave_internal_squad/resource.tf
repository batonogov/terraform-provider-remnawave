resource "remnawave_internal_squad" "premium" {
  name     = "Premium"
  inbounds = [join("", remnawave_config_profile.default.inbounds[*].uuid)]
}
