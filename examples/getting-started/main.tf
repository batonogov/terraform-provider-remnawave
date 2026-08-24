terraform {
  required_providers {
    remnawave = {
      source  = "batonogov/remnawave"
      version = "~> 1.7.1" # x-release-please-version
    }
  }
}

provider "remnawave" {
  # Configuration is read from REMNAWAVE_ENDPOINT and REMNAWAVE_API_TOKEN.
}

resource "remnawave_user" "quickstart" {
  username               = var.username
  expire_at              = var.expire_at
  traffic_limit_bytes    = 10737418240 # 10 GiB
  traffic_limit_strategy = "MONTH"
  description            = "Managed by Terraform"
}

variable "username" {
  type        = string
  description = "Username for the quick-start Remnawave user."
  default     = "terraform-quickstart"
}

variable "expire_at" {
  type        = string
  description = "Expiration timestamp for the quick-start user."
  default     = "2030-01-01T00:00:00.000Z"
}

output "subscription_url" {
  description = "Subscription URL created for the quick-start user."
  value       = remnawave_user.quickstart.subscription_url
  sensitive   = true
}