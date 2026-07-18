package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHostBulkActionResource_EnableDisable verifies the host_bulk_action
// resource can enable and disable a set of hosts.
func TestAccHostBulkActionResource_EnableDisable(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("host-bulk-enable", "VLESS_HOST_BULK_ENABLE") + `
resource "remnawave_host" "test" {
  remark  = "bulk-test"
  address = "127.0.0.1"
  port    = 443
  inbound = {
    config_profile_uuid         = remnawave_config_profile.profile.uuid
    config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
  }
  is_disabled = true
}

resource "remnawave_host_bulk_action" "enable" {
  action   = "enable"
  uuids    = [remnawave_host.test.uuid]
  triggers = { init = "1" }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_host_bulk_action.enable", "id"),
					resource.TestCheckResourceAttr("remnawave_host_bulk_action.enable", "action", "enable"),
				),
			},
		},
	})
}

// TestAccHostBulkActionResource_Delete verifies the bulk delete action.
func TestAccHostBulkActionResource_Delete(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("host-bulk-delete", "VLESS_HOST_BULK_DELETE") + `
resource "remnawave_host" "test" {
  remark  = "bulk-delete-test"
  address = "127.0.0.2"
  port    = 443
  inbound = {
    config_profile_uuid         = remnawave_config_profile.profile.uuid
    config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
  }
}

resource "remnawave_host_bulk_action" "delete" {
  action   = "delete"
  uuids    = [remnawave_host.test.uuid]
  triggers = { init = "1" }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_host_bulk_action.delete", "id"),
					resource.TestCheckResourceAttr("remnawave_host_bulk_action.delete", "action", "delete"),
				),
			},
		},
	})
}
