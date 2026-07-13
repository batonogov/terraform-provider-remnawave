data "remnawave_users" "all" {}

output "active_usernames" {
  value = [
    for user in data.remnawave_users.all.users : user.username
    if user.status == "ACTIVE"
  ]
}
