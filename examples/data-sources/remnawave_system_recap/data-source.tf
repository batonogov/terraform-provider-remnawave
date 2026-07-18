data "remnawave_system_recap" "current" {}

output "recap" {
  value = data.remnawave_system_recap.current
}
