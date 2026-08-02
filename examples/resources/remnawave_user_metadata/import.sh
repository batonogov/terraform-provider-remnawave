# Remnawave 3.0+ uses the numeric user ID.
terraform import remnawave_user_metadata.info 42

# Remnawave 2.x uses the user UUID.
terraform import remnawave_user_metadata.info 00000000-0000-0000-0000-000000000000
