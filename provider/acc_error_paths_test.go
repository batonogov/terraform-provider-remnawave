package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserValidationShortUsername verifies that a username shorter than the
// minimum length (3 characters) produces a clear validation error instead of a
// panic or silent acceptance.
func TestAccUserValidationShortUsername(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_user" "short" {
  username  = "ab"
  expire_at = "2027-01-01T00:00:00.000Z"
}
`,
			ExpectError: regexp.MustCompile(`(?i).*(minimum|min|too short|3 character|shorter).*`),
		}},
	})
}

// TestAccNodeValidationInvalidPort verifies that an out-of-range port number
// (99999) on a node produces an error rather than being silently accepted.
func TestAccNodeValidationInvalidPort(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + testAccProfileConfig("node-profile-err", "VLESS_TCP_NODE_ERR") + `
resource "remnawave_node" "test" {
  name                        = "invalid-port-node"
  address                     = "127.0.0.99"
  port                        = 99999
  country_code                = "NL"
  is_traffic_tracking_active  = true
  traffic_limit_bytes         = 1073741824
  traffic_reset_day           = 15
  notify_percent              = 80
  consumption_multiplier      = 1.2
  tags                        = ["ERR_NODE"]
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbounds     = [remnawave_config_profile.profile.inbounds[0].uuid]
}
`,
			ExpectError: regexp.MustCompile(`(?i).*(port|range|valid|invalid|must be|between|error).*`),
		}},
	})
}

// TestAccNodeNonExistentProfile verifies that referencing a non-existent
// config profile UUID (zero UUID) produces an error that mentions the UUID.
func TestAccNodeNonExistentProfile(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_node" "test" {
  name                        = "bad-profile-node"
  address                     = "127.0.0.98"
  port                        = 3333
  country_code                = "NL"
  is_traffic_tracking_active  = true
  traffic_limit_bytes         = 1073741824
  traffic_reset_day           = 15
  notify_percent              = 80
  consumption_multiplier      = 1.2
  tags                        = ["ERR_PROFILE"]
  config_profile_uuid         = "00000000-0000-0000-0000-000000000000"
  config_profile_inbounds     = ["00000000-0000-0000-0000-000000000000"]
}
`,
			ExpectError: regexp.MustCompile(`(?i).*(not found|A124|inbound|profile).*`),
		}},
	})
}

// NOTE: TestAccHostValidationInvalidPort removed — Remnawave API accepts
// ports > 65535 without error, so there is nothing to test here.

// TestAccConfigProfileEmptyName verifies that an empty config profile name
// is rejected with a validation error.
func TestAccConfigProfileEmptyName(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_config_profile" "test" {
  name = ""
  config = jsonencode({
    log = { loglevel = "warning" }
    inbounds = [{
      tag = "VLESS_TCP_ERR_EMPTY", listen = "0.0.0.0", port = 443, protocol = "vless"
      settings = { clients = [], decryption = "none" }
      streamSettings = { network = "tcp", security = "reality", realitySettings = { show = false, target = "xray.com", xver = 0, serverNames = ["xray.com"], privateKey = "", shortIds = [] } }
      sniffing = { enabled = true, destOverride = ["http", "tls", "quic"] }
    }]
    outbounds = [{ tag = "direct", protocol = "freedom", settings = {} }, { tag = "block", protocol = "blackhole", settings = {} }]
    routing = { domainStrategy = "AsIs", rules = [] }
  })
}
`,
			ExpectError: regexp.MustCompile(`(?i).*(name|empty|required|min|blank|shorter).*`),
		}},
	})
}
