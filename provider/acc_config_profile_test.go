package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccConfigProfileResource tests create → read → update → destroy.
func TestAccConfigProfileResource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_config_profile" "test" {
  name = "test-profile-acc-1"
  config = jsonencode({
    log      = { loglevel = "warning" }
    inbounds = [{
      tag      = "VLESS_TCP_REALITY_ACC1"
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
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_config_profile.test", "name", "test-profile-acc-1"),
					resource.TestCheckResourceAttrSet("remnawave_config_profile.test", "uuid"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_config_profile" "test" {
  name = "test-profile-renamed"
  config = jsonencode({
    log      = { loglevel = "warning" }
    inbounds = [{
      tag      = "VLESS_TCP_REALITY_ACC1"
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
`,
				Check: resource.TestCheckResourceAttr("remnawave_config_profile.test", "name", "test-profile-renamed"),
			},
			{
				ResourceName:            "remnawave_config_profile.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
				ImportStateIdFunc:       resourceUUIDImportStateID("remnawave_config_profile.test"),
			},
		},
	})
}

// TestAccConfigProfilesDataSource tests the config_profiles data source after creating a profile.
func TestAccConfigProfilesDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_config_profile" "test" {
  name = "ds-test-profile-4"
  config = jsonencode({
    log      = { loglevel = "warning" }
    inbounds = [{
      tag = "VLESS_TCP_REALITY_ACC4", listen = "0.0.0.0", port = 443, protocol = "vless"
      settings = { clients = [], decryption = "none" }
      streamSettings = { network = "tcp", security = "reality", realitySettings = { show = false, target = "xray.com", xver = 0, serverNames = ["xray.com"], privateKey = "", shortIds = [] } }
      sniffing = { enabled = true, destOverride = ["http", "tls", "quic"] }
    }]
    outbounds = [{ tag = "direct", protocol = "freedom", settings = {} }, { tag = "block", protocol = "blackhole", settings = {} }]
    routing = { domainStrategy = "AsIs", rules = [] }
  })
}

data "remnawave_config_profiles" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_config_profiles.all", "config_profiles.#"),
				),
			},
		},
	})
}
