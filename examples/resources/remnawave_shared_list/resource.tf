resource "remnawave_shared_list" "private_ranges" {
  name = "private_ranges"

  config = jsonencode({
    type  = "ipList"
    items = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  })
}

# Remnawave 3.4+: names may contain slash-separated segments.
resource "remnawave_shared_list" "grouped_ranges" {
  name = "terraform/bogons"

  config = jsonencode({
    type  = "ipList"
    items = ["100.64.0.0/10", "192.0.2.0/24"]
  })
}
