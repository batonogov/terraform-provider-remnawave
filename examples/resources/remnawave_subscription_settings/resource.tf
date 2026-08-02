resource "remnawave_subscription_settings" "main" {
  randomize_hosts        = true
  is_show_custom_remarks = true

  custom_response_headers = jsonencode({
    X-Managed-By = "terraform"
  })
}
