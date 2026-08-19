resource "remnawave_node_integration" "environment" {
  name        = "Common node environment"
  description = "Environment values injected into assigned nodes"

  config = jsonencode({
    environmentVariables = {
      XRAY_LOCATION_ASSET = "/usr/local/share/xray"
    }
  })

  # Force-restart nodes using this integration after an update.
  restart_nodes_on_update = true
}
