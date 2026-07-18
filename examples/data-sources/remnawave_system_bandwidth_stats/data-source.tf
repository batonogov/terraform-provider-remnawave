data "remnawave_system_bandwidth_stats" "current" {}

output "system_bandwidth" {
  value = data.remnawave_system_bandwidth_stats.current
}
