resource "remnawave_api_token" "ci" {
  name            = "ci-deploy"
  expires_in_days = 30
  scopes          = ["*"]
}
