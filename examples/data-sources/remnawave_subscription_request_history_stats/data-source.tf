data "remnawave_subscription_request_history_stats" "stats" {}

output "sub_history_stats" {
  value = data.remnawave_subscription_request_history_stats.stats
}
