package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDuplicateSnippetNameConflict verifies that creating two snippets with
// the same name produces a status-only bad-request error from the backend.
//
// Covers #118 (409 Conflict for duplicate snippet name).
func TestAccDuplicateSnippetNameConflict(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_snippet" "first" {
  name    = "dup-snippet-conflict"
  snippet = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads"] }])
}

resource "remnawave_snippet" "second" {
  name    = "dup-snippet-conflict"
  snippet = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads"] }])
}
`,
				ExpectError: regexp.MustCompile(`request failed: status 400`),
			},
		},
	})
}
