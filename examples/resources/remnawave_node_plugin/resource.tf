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

# Requires Remnawave 3.3.1 or later for torrentBlocker.rulePlacement.
resource "remnawave_node_plugin" "torrent_blocker" {
  name = "Torrent Blocker"

  plugin_config = jsonencode({
    torrentBlocker = {
      enabled = true
      # Position of the rule injected into the routing.rules array (0-1000).
      rulePlacement = 0
    }
  })
}
