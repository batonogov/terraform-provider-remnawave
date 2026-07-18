data "remnawave_system_health" "current" {}

output "health" {
  value = data.remnawave_system_health.current
}
