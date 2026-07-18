data "remnawave_hwid_stats" "current" {}

output "hwid_stats" {
  value = data.remnawave_hwid_stats.current
}
