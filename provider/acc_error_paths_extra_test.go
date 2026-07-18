package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDuplicateUsernameConflict verifies that creating two users with
// the same username produces a clear error (409 Conflict from backend).
func TestAccDuplicateUsernameConflict(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "first" {
  username            = "dup-conflict-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_user" "second" {
  username            = "dup-conflict-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
`,
				ExpectError: regexp.MustCompile(`(?i)(conflict|already exists|409|duplicate|error)`),
			},
		},
	})
}

// TestAccUserImportNotFound verifies that importing a non-existent user UUID
// produces an error. The exact error message depends on backend version.
func TestAccUserImportNotFound(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:            providerCfg,
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "remnawave_user.ghost",
				ImportStateId:     "00000000-0000-0000-0000-000000000000",
				ExpectError:       regexp.MustCompile(`(?i)(not found|404|error|failed)`),
			},
		},
	})
}

// TestAccNodeValidationInvalidPortExtended verifies that an out-of-range port
// is caught by plan-time validators (added in #111).
func TestAccNodeValidationInvalidPortExtended(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("node-port-err", "VLESS_NODE_PORT_ERR") + `
resource "remnawave_node" "test" {
  name                 = "invalid-port-extended"
  address              = "127.0.0.99"
  port                 = 99999
  config_profile_uuid  = remnawave_config_profile.profile.uuid
  config_profile_inbounds = [remnawave_config_profile.profile.inbounds[0].uuid]
}
`,
				ExpectError: regexp.MustCompile(`(?i)(invalid|between|value|range|must)`),
			},
		},
	})
}
