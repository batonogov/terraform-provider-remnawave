data "remnawave_bandwidth_realtime" "current" {}

output "realtime_bandwidth" {
  value = data.remnawave_bandwidth_realtime.current
}
