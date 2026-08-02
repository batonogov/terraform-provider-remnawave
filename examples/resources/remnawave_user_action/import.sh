# Remnawave 3.0+ uses <numeric_user_id>:<action>.
terraform import remnawave_user_action.extend "42:extend_expiration"

# Remnawave 2.x uses <user_uuid>:<action>.
terraform import remnawave_user_action.reset "00000000-0000-0000-0000-000000000000:reset_traffic"
