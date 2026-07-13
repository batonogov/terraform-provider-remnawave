data "remnawave_system_stats" "current" {
  tz = "Europe/Moscow"
}

output "online_users" {
  value = data.remnawave_system_stats.current.online_now
}
