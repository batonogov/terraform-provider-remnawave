data "remnawave_bandwidth_stats" "monthly" {
  start = "2026-01-01T00:00:00+03:00"
  end   = "2026-01-31T23:59:59+03:00"
}

output "monthly_bandwidth" {
  value = data.remnawave_bandwidth_stats.monthly
}
