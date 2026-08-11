resource "remnawave_snippet" "routing_rules" {
  name = "block-ads"
  snippet = jsonencode([
    {
      type        = "field"
      outboundTag = "block"
      domain      = ["geosite:category-ads"]
    }
  ])

  # Remnawave 3.2.3+: restart affected nodes after update or deletion.
  # sync_nodes_on_change = true
}
