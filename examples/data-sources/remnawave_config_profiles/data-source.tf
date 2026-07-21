data "remnawave_config_profiles" "all" {}

output "profiles" {
  value = data.remnawave_config_profiles.all.config_profiles
}
