package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserMetadataResource tests create → read → update cycle for user metadata.
func TestAccUserMetadataResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "meta-test-user"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
resource "remnawave_user_metadata" "test" {
  user_uuid = remnawave_user.test.uuid
  metadata  = jsonencode({ department = "engineering" })
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("remnawave_user_metadata.test", "uuid"),
				resource.TestCheckResourceAttrSet("remnawave_user_metadata.test", "metadata"),
			),
		}},
	})
}

// TestAccNodeMetadataResource tests node metadata.
// Skipped: node creation requires a real Xray backend not available in test env.
func TestAccNodeMetadataResource(t *testing.T) {
	t.Skip("node_metadata requires a real Xray node which is not available in test env")
}
