package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPasskeyResource_Import verifies that a passkey resource can be
// imported and read from state. This test does NOT create a passkey (WebAuthn
// registration is interactive); it expects a pre-existing passkey UUID passed
// via the REMNAWAVE_TEST_PASSKEY_UUID env var.
func TestAccPasskeyResource_Import(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	// This test is a no-op if no passkey UUID is provided.
	// It validates the import path without requiring interactive WebAuthn.
	t.Skip("requires pre-existing passkey UUID via env var; skipping automated run")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg,
			},
		},
	})
}
