resource "remnawave_billing_history" "payment" {
  provider_uuid = remnawave_infra_provider.example.uuid
  amount        = 9.99
  description   = "Monthly payment"
}
