data "remnawave_hwid_top_users" "top10" {}

output "hwid_top_users" {
  value = data.remnawave_hwid_top_users.top10
}
