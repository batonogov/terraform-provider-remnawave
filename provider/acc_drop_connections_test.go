package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDropConnectionsResource_ByUserUUID verifies the V2 request reaches
// the backend and returns a status-only not-found diagnostic. The compose
// fixture intentionally has no connected Xray node, so a successful connection
// drop is impossible.
func TestAccDropConnectionsResource_ByUserUUID(t *testing.T) {
	testAccPreCheck(t)

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
				ExpectError: regexp.MustCompile(`request failed: status 404`),
			},
		},
	})
}
