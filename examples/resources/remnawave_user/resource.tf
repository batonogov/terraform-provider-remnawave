resource "remnawave_user" "example" {
  username               = "john-doe"
  expire_at              = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes    = 10737418240 # 10 GiB
  traffic_limit_strategy = "MONTH"
  description            = "Managed by Terraform"
  tag                    = "CUSTOMER"
  hwid_device_limit      = 3
}
