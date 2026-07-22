terraform {
  required_providers {
    remnawave = {
      source  = "batonogov/remnawave"
      version = "~> 0.5.0"
    }
  }
}

# API token authentication (recommended)
provider "remnawave" {
  endpoint  = "https://panel.example.com"
  api_token = var.remnawave_api_token

  # Optional authentication for an outer reverse-proxy gate:
  # custom_headers = {
  #   Cookie = var.remnawave_gateway_cookie # cookie_name=cookie_value
  # }
}

# Username/password authentication is also supported
# provider "remnawave" {
#   endpoint = "https://panel.example.com"
#   username = "admin"
#   password = var.remnawave_password
# }

variable "remnawave_api_token" {
  type      = string
  sensitive = true
}

# Create a VPN user
resource "remnawave_user" "example" {
  username               = "john-doe"
  expire_at              = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes    = 10737418240 # 10 GB
  traffic_limit_strategy = "MONTH"
  description            = "Example user managed by Terraform"
}

# List all nodes
data "remnawave_nodes" "all" {}

output "subscription_url" {
  value = remnawave_user.example.subscription_url
}

output "nodes" {
  value = data.remnawave_nodes.all.nodes
}
