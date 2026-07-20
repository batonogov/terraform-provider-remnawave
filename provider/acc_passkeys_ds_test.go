package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPasskeysDataSource verifies the passkeys data source can query
// /api/passkeys without an unmarshal error. No passkeys are registered in
// the test environment (registration is an interactive WebAuthn ceremony),
// so the result will be an empty list — but the data source must not crash.
//
// Passkeys require admin JWT auth (ROLE.ADMIN), not an API token (ROLE.API).
// This test is skipped when REMNAWAVE_API_TOKEN is set.
//
// Covers #117 (passkeys) and verifies the fix for #142.
func TestAccPasskeysDataSource(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv(envAPIToken) != "" {
		t.Skip("passkeys require admin JWT auth, skipping under API token")
	}
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
					// No passkeys in test env — list should be empty or absent.
					resource.TestCheckResourceAttr("data.remnawave_passkeys.all", "passkeys.#", "0"),
				),
			},
		},
	})
}
