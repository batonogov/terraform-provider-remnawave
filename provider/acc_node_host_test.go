package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// isBackend2_7 returns true when REMNAWAVE_VERSION env is set to a 2.7.x tag.
// Used to skip 2.8-only fields (xhttp_extra_params, mux_params, sockopt_params,
// final_mask) that the 2.7 API silently strips from requests but does not echo
// back, causing Terraform "inconsistent result after apply" errors.
func isBackend2_7() bool {
	v := os.Getenv("REMNAWAVE_VERSION")
	return strings.HasPrefix(v, "2.7.")
}

// hostV28Fields returns the 2.8-only JSON fields block for remnawave_host, or
// an empty string when running against a 2.7.x backend.
func hostV28Fields(mode, muxEnabled, tfo, finalMaskEnabled string) string {
	if isBackend2_7() {
		return ""
	}
	return fmt.Sprintf(`  xhttp_extra_params = jsonencode({ mode = %q })
  mux_params         = jsonencode({ enabled = %s })
  sockopt_params     = jsonencode({ tcpFastOpen = %s })
  final_mask         = jsonencode({ enabled = %s })
`, mode, muxEnabled, tfo, finalMaskEnabled)
}

func TestAccNodeResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("node-profile", "VLESS_TCP_NODE_ACC") + `
resource "remnawave_node" "test" {
  name                        = "terraform-node"
  address                     = "127.0.0.10"
  port                        = 2222
  country_code                = "NL"
  is_traffic_tracking_active  = true
  traffic_limit_bytes         = 1073741824
  traffic_reset_day           = 15
  notify_percent              = 80
  consumption_multiplier      = 1.2
  tags                        = ["ACC_NODE"]
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbounds     = [remnawave_config_profile.profile.inbounds[0].uuid]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_node.test", "uuid"),
					resource.TestCheckResourceAttr("remnawave_node.test", "name", "terraform-node"),
					resource.TestCheckResourceAttr("remnawave_node.test", "country_code", "NL"),
					resource.TestCheckResourceAttr("remnawave_node.test", "consumption_multiplier", "1.2"),
					resource.TestCheckResourceAttr("remnawave_node.test", "config_profile_inbounds.#", "1"),
				),
			},
			{
				Config: providerCfg + testAccProfileConfig("node-profile", "VLESS_TCP_NODE_ACC") + `
resource "remnawave_node" "test" {
  name                        = "terraform-node-updated"
  address                     = "127.0.0.10"
  port                        = 2222
  country_code                = "DE"
  is_traffic_tracking_active  = true
  consumption_multiplier      = 2.0
  tags                        = ["ACC_NODE", "UPDATED"]
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbounds     = [remnawave_config_profile.profile.inbounds[0].uuid]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node.test", "name", "terraform-node-updated"),
					resource.TestCheckResourceAttr("remnawave_node.test", "country_code", "DE"),
					resource.TestCheckResourceAttr("remnawave_node.test", "tags.#", "2"),
				),
			},
			{
				ResourceName:                         "remnawave_node.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"updated_at", "last_status_change"},
				ImportStateIdFunc:                    resourceUUIDImportStateID("remnawave_node.test"),
			},
		},
	})
}

func TestAccHostResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("host-profile", "VLESS_TCP_HOST_ACC") + fmt.Sprintf(`
resource "remnawave_host" "test" {
  remark                      = "terraform-host"
  address                     = "host.example.com"
  port                        = 443
  sni                         = "host.example.com"
  security_layer              = "TLS"
  override_sni_from_address   = true
  keep_sni_blank              = false
  vless_route_id              = 7
%s
  xray_json_template_uuid     = remnawave_subscription_template.host.uuid
  exclude_from_subscription_types = ["MIHOMO", "SINGBOX"]
  tags                        = ["ACC_HOST"]
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
}

resource "remnawave_subscription_template" "host" {
  name          = "host-acceptance-template"
  template_type = "XRAY_JSON"
}
`, hostV28Fields("auto", "true", "true", "false")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_host.test", "uuid"),
					resource.TestCheckResourceAttr("remnawave_host.test", "remark", "terraform-host"),
					resource.TestCheckResourceAttr("remnawave_host.test", "address", "host.example.com"),
					resource.TestCheckResourceAttr("remnawave_host.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("remnawave_host.test", "vless_route_id", "7"),
					resource.TestCheckResourceAttr("remnawave_host.test", "exclude_from_subscription_types.#", "2"),
				),
			},
			{
				Config: providerCfg + testAccProfileConfig("host-profile", "VLESS_TCP_HOST_ACC") + fmt.Sprintf(`
resource "remnawave_host" "test" {
  remark                      = "terraform-host-updated"
  address                     = "updated.example.com"
  port                        = 8443
  sni                         = "updated.example.com"
  security_layer              = "TLS"
  is_hidden                   = true
  override_sni_from_address   = true
  keep_sni_blank              = false
  vless_route_id              = 8
%s
  xray_json_template_uuid     = remnawave_subscription_template.host.uuid
  exclude_from_subscription_types = ["CLASH"]
  tags                        = ["ACC_HOST", "UPDATED"]
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
}

resource "remnawave_subscription_template" "host" {
  name          = "host-acceptance-template"
  template_type = "XRAY_JSON"
}
`, hostV28Fields("packet-up", "false", "false", "true")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_host.test", "remark", "terraform-host-updated"),
					resource.TestCheckResourceAttr("remnawave_host.test", "port", "8443"),
					resource.TestCheckResourceAttr("remnawave_host.test", "is_hidden", "true"),
					resource.TestCheckResourceAttr("remnawave_host.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("remnawave_host.test", "vless_route_id", "8"),
					resource.TestCheckResourceAttr("remnawave_host.test", "exclude_from_subscription_types.#", "1"),
				),
			},
			{
				ResourceName:      "remnawave_host.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"xhttp_extra_params", "mux_params", "sockopt_params", "final_mask",
				},
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    resourceUUIDImportStateID("remnawave_host.test"),
			},
		},
	})
}

func testAccProfileConfig(name, inboundTag string) string {
	return fmt.Sprintf(`
resource "remnawave_config_profile" "profile" {
  name = %q
  config = jsonencode({
    log = { loglevel = "warning" }
    inbounds = [{
      tag      = %q
      listen   = "0.0.0.0"
      port     = 443
      protocol = "vless"
      settings = { clients = [], decryption = "none" }
      streamSettings = {
        network  = "tcp"
        security = "reality"
        realitySettings = {
          show        = false
          target      = "xray.com"
          xver        = 0
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
`, name, inboundTag)
}

func resourceUUIDImportStateID(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		return state.RootModule().Resources[resourceName].Primary.Attributes["uuid"], nil
	}
}

// resourceAttrImportStateID returns an ImportStateIdFunc that reads the given
// attribute from the Terraform state. Useful for resources whose import key is
// not "uuid" (e.g. remnawave_snippet uses "name", hwid_device uses "id").
func resourceAttrImportStateID(resourceName, attr string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		return state.RootModule().Resources[resourceName].Primary.Attributes[attr], nil
	}
}

// staticImportStateID returns an ImportStateIdFunc that always returns the
// given ID string. Used by singleton resources (panel_settings,
// subscription_settings) whose import ID is always "settings".
func staticImportStateID(id string) resource.ImportStateIdFunc {
	return func(*terraform.State) (string, error) { return id, nil }
}
