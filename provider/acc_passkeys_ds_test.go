package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPasskeysDataSource verifies the remnawave_passkeys data source reads
// the list of WebAuthn passkeys for the current admin user. WebAuthn
// registration is interactive, so the list may be empty — what we assert is
// that the data source returns no error and the passkeys list attribute is
// present in state.
//
// The passkeys endpoint requires admin JWT auth (username/password), not an
// API token — the test is skipped when api_token auth is detected.
//
// Covers #117 for remnawave_passkeys.
func TestAccPasskeysDataSource(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv(envAPIToken) != "" {
		t.Skip("passkeys data source requires admin JWT — skipped when using api_token auth")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// No passkey registered in CI → expect an empty list.
				// The data source must still populate the attribute.
				Config: providerCfg + `
data "remnawave_passkeys" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remnawave_passkeys.all", "passkeys.#", "0"),
				),
			},
		},
	})
}
