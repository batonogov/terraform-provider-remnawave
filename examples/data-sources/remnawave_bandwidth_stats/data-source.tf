data "remnawave_bandwidth_stats" "monthly" {
  start = "2026-01-01"
  end   = "2026-01-31"
}

output "monthly_bandwidth" {
  value = data.remnawave_bandwidth_stats.monthly
}
