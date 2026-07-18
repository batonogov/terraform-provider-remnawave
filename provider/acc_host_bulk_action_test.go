package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHostBulkActionResource_Disable verifies the host_bulk_action
// resource can disable a set of hosts.
// Note: we don't test "enable" because it causes a refresh-plan diff
// (is_disabled changes from true→false outside Terraform).
// We don't test "delete" because the host is destroyed by the bulk action,
// causing a 404 during Terraform destroy cleanup.
func TestAccHostBulkActionResource_Disable(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + testAccProfileConfig("host-bulk-disable", "VLESS_HOST_BULK_DISABLE") + `
resource "remnawave_host" "test" {
  remark                      = "bulk-disable-test"
  address                     = "127.0.0.3"
  port                        = 443
  config_profile_uuid         = remnawave_config_profile.profile.uuid
  config_profile_inbound_uuid = remnawave_config_profile.profile.inbounds[0].uuid
}

resource "remnawave_host_bulk_action" "disable" {
  action   = "disable"
  uuids    = [remnawave_host.test.uuid]
  triggers = { init = "1" }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_host_bulk_action.disable", "id"),
					resource.TestCheckResourceAttr("remnawave_host_bulk_action.disable", "action", "disable"),
				),
			},
		},
	})
}
