package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserMetadataResource tests create → read → update cycle for user metadata.
func TestAccUserMetadataResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "meta-test-user"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
resource "remnawave_user_metadata" "test" {
  user_uuid = remnawave_user.test.uuid
  metadata  = jsonencode({ department = "engineering" })
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_user_metadata.test", "uuid"),
				resource.TestCheckResourceAttr("remnawave_user_metadata.test", "metadata", `{"department":"engineering"}`),
			),
		}},
	})
}

// TestAccNodeMetadataResource tests create → read → update cycle for node metadata.
func TestAccNodeMetadataResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_config_profile" "test" {
  name = "node-meta-test-profile"
  config = jsonencode({
    log      = { loglevel = "warning" }
    inbounds = [{
      tag      = "VLESS_TCP_META"
      listen   = "0.0.0.0"
      port     = 443
      protocol = "vless"
      settings = { clients = [], decryption = "none" }
      streamSettings = {
        network  = "tcp"
        security = "reality"
        realitySettings = {
          show       = false
          target     = "xray.com"
          xver       = 0
          serverNames = ["xray.com"]
          privateKey  = ""
          shortIds    = []
        }
      }
      sniffing = { enabled = true, destOverride = ["http", "tls", "quic"] }
    }]
    outbounds = [
      { tag = "direct", protocol = "freedom", settings = {} },
      { tag = "block", protocol = "blackhole", settings = {} }
    ]
    routing = { domainStrategy = "AsIs", rules = [] }
  })
}

resource "remnawave_node" "test" {
  name                = "meta-test-node"
  address             = "127.0.0.1"
  config_profile_uuid = remnawave_config_profile.test.uuid
}

resource "remnawave_node_metadata" "test" {
  node_uuid = remnawave_node.test.uuid
  metadata  = jsonencode({ location = "us-east-1", tag = "production" })
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_node_metadata.test", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_node_metadata.test", "node_uuid"),
			),
		}},
	})
}
