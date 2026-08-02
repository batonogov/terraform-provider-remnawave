resource "remnawave_external_squad" "standard" {
  name = "Standard"

  # Remnawave 3.0+ response-header configuration. These fields remain as
  # no-ops in state when the provider is connected to a 2.x panel.
  response_headers_add = jsonencode({
    X-Squad = "standard"
  })
  response_headers_remove = jsonencode([
    "X-Legacy-Header",
  ])
}
