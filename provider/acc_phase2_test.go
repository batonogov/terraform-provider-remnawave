package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExternalSquadResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_external_squad" "test" { name = "test-ext-squad" }
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_external_squad.test", "name", "test-ext-squad"),
					resource.TestCheckResourceAttrSet("remnawave_external_squad.test", "uuid"),
				),
			},
			{
				ResourceName:                         "remnawave_external_squad.test",
				ImportState:                          true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"updated_at"},
				ImportStateIdFunc:                    resourceUUIDImportStateID("remnawave_external_squad.test"),
			},
		},
	})
}

func TestAccInternalSquadResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_internal_squad" "test" {
  name     = "test-int-squad"
  inbounds = []
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_internal_squad.test", "name", "test-int-squad"),
					resource.TestCheckResourceAttrSet("remnawave_internal_squad.test", "uuid"),
				),
			},
			{
				ResourceName:                         "remnawave_internal_squad.test",
				ImportState:                          true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"updated_at"},
				ImportStateIdFunc:                    resourceUUIDImportStateID("remnawave_internal_squad.test"),
			},
		},
	})
}

func TestAccSubscriptionTemplateResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_subscription_template" "test" {
  name          = "test-template"
  template_type = "XRAY_JSON"
  template_json = jsonencode({ log = { loglevel = "warning" } })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_subscription_template.test", "name", "test-template"),
					resource.TestCheckResourceAttr("remnawave_subscription_template.test", "template_type", "XRAY_JSON"),
					resource.TestCheckResourceAttrSet("remnawave_subscription_template.test", "template_json"),
					resource.TestCheckResourceAttrSet("remnawave_subscription_template.test", "uuid"),
				),
			},
			{
				ResourceName:                         "remnawave_subscription_template.test",
				ImportState:                          true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"updated_at"},
				ImportStateIdFunc:                    resourceUUIDImportStateID("remnawave_subscription_template.test"),
			},
		},
	})
}

func TestAccPanelSettingsResource(t *testing.T) {
	// Panel settings endpoint may require admin JWT (not API token).
	// Skip if 403 — this is a known panel restriction.
	testAccPreCheck(t)
	if os.Getenv(envAPIToken) != "" {
		t.Skip("panel_settings requires admin JWT, not API token — skipped when using api_token auth")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_panel_settings" "test" {
  branding_title = "My Panel"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_panel_settings.test", "id", "settings"),
					resource.TestCheckResourceAttr("remnawave_panel_settings.test", "branding_title", "My Panel"),
				),
			},
			{
				ResourceName:                         "remnawave_panel_settings.test",
				ImportState:                          true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"updated_at"},
				ImportStateIdFunc:                    staticImportStateID("settings"),
			},
		},
	})
}
