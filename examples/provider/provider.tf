terraform {
  required_providers {
    remnawave = {
      source  = "batonogov/remnawave"
      version = "~> 0.2.0"
    }
  }
}

variable "remnawave_endpoint" {
  type        = string
  description = "Base URL of the Remnawave panel."
}

variable "remnawave_api_token" {
  type        = string
  description = "API token created in the Remnawave panel."
  sensitive   = true
}

provider "remnawave" {
  endpoint  = var.remnawave_endpoint
  api_token = var.remnawave_api_token
}
