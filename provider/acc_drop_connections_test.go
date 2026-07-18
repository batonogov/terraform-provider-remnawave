package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDropConnectionsResource_ByUserUUID verifies the drop_connections
// resource using the V2 schema (drop_by + user_uuids).
// SKIPPED until PR #130 (drop_connections full schema) is merged — the old
// user_uuid attribute no longer works with Remnawave 2.8.0 backend.
func TestAccDropConnectionsResource_ByUserUUID(t *testing.T) {
	testAccPreCheck(t)
	t.Skip("depends on PR #130 (drop_connections V2 schema) being merged into main")

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "drop-conn-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_drop_connections" "test" {
  drop_by    = "user_uuids"
  user_uuids = [remnawave_user.test.uuid]
  triggers   = { init = "1" }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("remnawave_drop_connections.test", "id"),
				),
			},
		},
	})
}
