resource "remnawave_infra_provider" "example" {
  name = "Provider Co"
}

resource "remnawave_billing_node" "example" {
  provider_uuid   = remnawave_infra_provider.example.uuid
  next_billing_at = "2027-01-01T00:00:00.000Z"
}
