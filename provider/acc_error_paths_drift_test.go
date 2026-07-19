package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDuplicateSnippetNameConflict verifies that creating two snippets with
// the same name produces a clear error (409 Conflict from backend).
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
				ExpectError: regexp.MustCompile(`(?i)(conflict|already exists|409|duplicate|error)`),
			},
		},
	})
}

// TestAccSnippetImportNotFound verifies that importing a non-existent snippet
// name produces an error. Snippet is keyed by name (not UUID), making its
// import path distinct from most other resources.
//
// Covers #118 (ImportState error path for snippet).
func TestAccSnippetImportNotFound(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:        providerCfg,
				ImportState:   true,
				ResourceName:  "remnawave_snippet.ghost",
				ImportStateId: "nonexistent-snippet-name-xyz",
				ExpectError:   regexp.MustCompile(`(?i)(not found|404|error|failed)`),
			},
		},
	})
}

// TestAccHostImportNotFound verifies that importing a non-existent host UUID
// produces an error. Exercises the host ImportState → Read path where
// isNotFound() must surface the 404 rather than silently dropping the resource.
//
// Covers #118 (ImportState 404 drift for host).
func TestAccHostImportNotFound(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:        providerCfg,
				ImportState:   true,
				ResourceName:  "remnawave_host.ghost",
				ImportStateId: "00000000-0000-0000-0000-000000000000",
				ExpectError:   regexp.MustCompile(`(?i)(not found|404|error|failed)`),
			},
		},
	})
}

// TestAccNodeImportNotFound verifies that importing a non-existent node UUID
// produces an error. Exercises the node ImportState → Read path where
// isNotFound() must surface the 404 rather than silently dropping the resource.
//
// Covers #118 (ImportState 404 drift for node).
func TestAccNodeImportNotFound(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:        providerCfg,
				ImportState:   true,
				ResourceName:  "remnawave_node.ghost",
				ImportStateId: "00000000-0000-0000-0000-000000000000",
				ExpectError:   regexp.MustCompile(`(?i)(not found|404|error|failed)`),
			},
		},
	})
}

// TestAccConfigProfileImportNotFound verifies that importing a non-existent
// config profile UUID surfaces a 404 error. Exercises the config profile
// ImportState → Read path where isNotFound() removes the resource from state.
//
// Covers #118 (ImportState 404 drift for config_profile).
func TestAccConfigProfileImportNotFound(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:        providerCfg,
				ImportState:   true,
				ResourceName:  "remnawave_config_profile.ghost",
				ImportStateId: "00000000-0000-0000-0000-000000000000",
				ExpectError:   regexp.MustCompile(`(?i)(not found|404|error|failed)`),
			},
		},
	})
}
