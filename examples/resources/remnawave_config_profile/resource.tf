resource "remnawave_config_profile" "default" {
  name = "default-profile"
  config = jsonencode({
    inbounds = [{
      tag      = "vless-in"
      listen   = "0.0.0.0"
      port     = 443
      protocol = "vless"
      settings = {
        decryption = "none"
        clients    = []
      }
      streamSettings = {
        network  = "tcp"
        security = "tls"
      }
    }]
    routing = { rules = [] }
  })
}
