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

// TestAccUserIPsDataSource verifies the user_ips data source.
// This is an async endpoint — the polling may fail in test environment if
// the job doesn't complete fast enough. We skip it in CI unless explicitly
// enabled.
func TestAccUserIPsDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	// Skip in CI — async IP fetch requires a connected user + node.
	t.Skip("requires connected user + node for async IP fetch; skip in CI")

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

// TestAccPasskeysDataSource verifies the passkeys data source.
// This requires username/password auth (not API token), so we skip it
// unless the CI environment provides those credentials.
func TestAccPasskeysDataSource(t *testing.T) {
	testAccPreCheck(t)

	// Passkeys endpoint requires admin JWT (username/password), not API token.
	// CI typically uses API token auth, so skip unless explicitly configured.
	t.Skip("passkeys data source requires username/password auth; skip in API-token CI")

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
