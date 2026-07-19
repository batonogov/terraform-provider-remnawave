package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserIPsDataSource verifies the remnawave_user_ips data source can be
// read against a real user. The user has no active connections in the test
// panel, so the returned IP list is expected to be empty — what we assert is
// that the data source returns no error and the ips attribute is set.
//
// Covers #117 for remnawave_user_ips.
func TestAccUserIPsDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "probe" {
  username            = "user-ips-ds-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

data "remnawave_user_ips" "current" {
  uuid = remnawave_user.probe.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_user_ips.current", "uuid"),
					// No active connections in test panel → ips list is empty.
					resource.TestCheckResourceAttr("data.remnawave_user_ips.current", "ips.#", "0"),
				),
			},
		},
	})
}
