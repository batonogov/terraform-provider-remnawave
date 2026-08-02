# Remnawave 3.0+ uses <numeric_user_id>:<hwid>.
terraform import remnawave_hwid_device.phone 42:device-fingerprint-abc123

# Remnawave 2.x uses <user_uuid>:<hwid>.
terraform import remnawave_hwid_device.phone 00000000-0000-0000-0000-000000000000:device-fingerprint-abc123
