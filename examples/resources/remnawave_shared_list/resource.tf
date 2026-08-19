resource "remnawave_shared_list" "private_ranges" {
  name = "private_ranges"

  config = jsonencode({
    type  = "ipList"
    items = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  })
}
