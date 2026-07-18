resource "remnawave_snippet" "routing_rules" {
  name = "block-ads"
  content = jsonencode({
    rules = [{
      type        = "field"
      outboundTag = "block"
      domain      = ["geosite:category-ads"]
    }]
  })
}
