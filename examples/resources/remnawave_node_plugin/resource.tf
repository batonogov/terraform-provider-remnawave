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

# Review the Remnawave torrent blocker documentation before enabling this
# plugin: it bans matching client IPs for blockDuration seconds.
resource "remnawave_node_plugin" "torrent_blocker" {
  name = "Torrent Blocker"

  plugin_config = jsonencode({
    torrentBlocker = {
      enabled       = false
      blockDuration = 3600
      ignoreLists = {
        # Plain addresses only; the backend rejects CIDR ranges here.
        ip = ["203.0.113.10"]
      }
      # Position of the injected routing rule (0-1000).
      # Requires Remnawave 3.3.1 or later.
      rulePlacement = 0
    }
  })
}
