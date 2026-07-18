package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHostTagsDataSource verifies that the host_tags data source returns
// a list of tags from the panel.
func TestAccHostTagsDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
data "remnawave_host_tags" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_host_tags.all", "tags.#"),
				),
			},
		},
	})
}

// TestAccUserIPsDataSource verifies that the user_ips data source can
// fetch connection IPs for a user. Since this is async and the user
// may not be connected, we only check the data source runs without error.
func TestAccUserIPsDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "ips-ds-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

data "remnawave_user_ips" "test" {
  uuid = remnawave_user.test.uuid
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_user_ips.test", "ips.#"),
				),
			},
		},
	})
}

// TestAccPasskeysDataSource verifies the passkeys data source returns a list
// of passkeys for the current admin. Requires username/password auth.
func TestAccPasskeysDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
data "remnawave_passkeys" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_passkeys.all", "passkeys.#"),
				),
			},
		},
	})
}
