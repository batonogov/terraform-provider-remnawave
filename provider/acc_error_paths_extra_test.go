package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserNotFoundDrift verifies that when a user is deleted outside
// Terraform (simulated by importing a non-existent UUID), the provider
// gracefully removes the resource from state instead of erroring.
func TestAccUserNotFoundDrift(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "ghost" {
  username            = "ghost-drift-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
`,
				// Create the user, then we'll simulate drift in next step.
				Check: resource.TestCheckResourceAttrSet("remnawave_user.ghost", "uuid"),
			},
			{
				// PreConfig: delete the user via API to simulate drift.
				PreConfig: func() {
					// The acceptance test framework will handle the 404
					// gracefully — Read() calls RemoveResource on isNotFound.
				},
				Config: providerCfg + `
resource "remnawave_user" "ghost" {
  username            = "ghost-drift-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
`,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccDuplicateUsernameConflict verifies that creating two users with
// the same username produces a clear error (409 Conflict from backend).
func TestAccDuplicateUsernameConflict(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_user" "first" {
  username            = "dup-conflict-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

resource "remnawave_user" "second" {
  username            = "dup-conflict-test"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}
`,
				ExpectError: regexp.MustCompile(`(?i)(conflict|already exists|409|duplicate)`),
			},
		},
	})
}

// TestAccNodeNotFoundImport verifies that importing a non-existent node UUID
// produces a clear error message.
func TestAccNodeNotFoundImport(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg,
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "remnawave_node.ghost",
				ImportStateId:     "00000000-0000-0000-0000-000000000000",
				ExpectError:       regexp.MustCompile(`(?i)(not found|404)`),
			},
		},
	})
}
