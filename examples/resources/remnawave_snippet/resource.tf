resource "remnawave_snippet" "routing_rules" {
  name = "block-ads"
  snippet = jsonencode([
    {
      type        = "field"
      outboundTag = "block"
      domain      = ["geosite:category-ads"]
    }
  ])
}
