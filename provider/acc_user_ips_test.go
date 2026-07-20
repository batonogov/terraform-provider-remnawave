package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserIPsDataSource verifies the user_ips data source can query the
// IP control endpoint without a 404. The fetch-ips job returns a result even
// if no IPs are found (the user simply has no active connections in a test
// environment), so we only assert that the data source returns successfully.
//
// Covers #117 (user_ips) and verifies the fix for #141.
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
				// The data source may return an empty list if the user has no
				// active connections — that's fine. We just need to confirm
				// no 404 or unmarshal error.
				ExpectError: nil,
			},
		},
	})
}
