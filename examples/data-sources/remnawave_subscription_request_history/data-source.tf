data "remnawave_subscription_request_history" "recent" {}

output "sub_history" {
  value = data.remnawave_subscription_request_history.recent
}
