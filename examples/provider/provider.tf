terraform {
  required_providers {
    remnawave = {
      source  = "batonogov/remnawave"
      version = "~> 0.6.0"
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

variable "remnawave_gateway_cookie" {
  type        = string
  description = "Complete cookie pair required by an optional outer reverse-proxy gate, for example cookie_name=cookie_value."
  sensitive   = true
  default     = null
}

provider "remnawave" {
  endpoint  = var.remnawave_endpoint
  api_token = var.remnawave_api_token

  # Optional authentication for an outer reverse-proxy gate:
  # custom_headers = {
  #   Cookie = var.remnawave_gateway_cookie # cookie_name=cookie_value
  # }
}
