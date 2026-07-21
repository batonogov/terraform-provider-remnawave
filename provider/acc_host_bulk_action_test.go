package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHostBulkActionResource_Enable verifies an idempotent host bulk action.
func TestAccHostBulkActionResource_Enable(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("host-bulk-enable", "VLESS_HOST_BULK_ENABLE") + `
resource "remnawave_host" "test" {
  remark                      = "bulk-enable-test"
  address                     = "127.0.0.3"
  port                        = 443
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
}

resource "remnawave_host_bulk_action" "enable" {
  action   = "enable"
  uuids    = [remnawave_host.test.uuid]
  triggers = { init = "1" }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_host_bulk_action.enable", "action", "enable"),
					resource.TestCheckResourceAttr("remnawave_host_bulk_action.enable", "uuids.#", "1"),
					resource.TestCheckResourceAttrSet("remnawave_host_bulk_action.enable", "id"),
				),
			},
		},
	})
}
