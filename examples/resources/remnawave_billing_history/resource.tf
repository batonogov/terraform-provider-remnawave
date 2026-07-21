resource "remnawave_billing_history" "payment" {
  provider_uuid = remnawave_infra_provider.example.uuid
  amount        = 9.99
  billed_at     = "2026-01-15T12:00:00.000Z"
}
