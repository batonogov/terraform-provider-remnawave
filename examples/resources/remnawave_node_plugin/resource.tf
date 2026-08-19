resource "remnawave_node_plugin" "pre_start" {
  name = "Pre-Start Cleanup"

  plugin_config = jsonencode({
    preStart = {
      enabled = true
      cleanupSockets = {
        enabled = true
        files   = ["/dev/shm/*.sock"]
      }
    }
  })
}
