package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSnippetResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_snippet" "test" {
  name    = "test-snippet-2"
  snippet = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads"] }])
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_snippet.test", "name", "test-snippet-2"),
				),
			},
			{
				ResourceName:            "remnawave_snippet.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
				ImportStateIdFunc:       resourceAttrImportStateID("remnawave_snippet.test", "name"),
			},
		},
	})
}

func TestAccNodePluginResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_node_plugin" "test" {
  name          = "test-plugin"
  plugin_config = jsonencode({
    sharedLists = []
    connectionDrop = {
      enabled      = false
      whitelistIps = []
    }
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_node_plugin.test", "name", "test-plugin"),
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.test", "uuid"),
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.test", "plugin_config"),
				),
			},
			{
				ResourceName:            "remnawave_node_plugin.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
				ImportStateIdFunc:       resourceUUIDImportStateID("remnawave_node_plugin.test"),
			},
		},
	})
}

func TestAccApiTokenResource(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv(envAPIToken) != "" {
		t.Skip("api_token resource requires admin JWT — skipped when using api_token auth")
	}
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet("remnawave_api_token.test", "uuid"),
		resource.TestCheckResourceAttrSet("remnawave_api_token.test", "token"),
		resource.TestCheckResourceAttr("remnawave_api_token.test", "name", "terraform-acceptance"),
	}
	// expire_at, expires_in_days, and scopes are 2.8.x-only — 2.7.x does not
	// return them in the token response.
	if !isBackend2_7() {
		checks = append(checks,
			resource.TestCheckResourceAttrSet("remnawave_api_token.test", "expire_at"),
			resource.TestCheckResourceAttr("remnawave_api_token.test", "expires_in_days", "2"),
			resource.TestCheckResourceAttr("remnawave_api_token.test", "scopes.#", "1"),
		)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_api_token" "test" {
  name            = "terraform-acceptance"
  expires_in_days = 2
  scopes          = ["*"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(checks...),
			},
			{
				ResourceName:            "remnawave_api_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "updated_at"},
				ImportStateIdFunc:       resourceUUIDImportStateID("remnawave_api_token.test"),
			},
		},
	})
}

func TestAccInfraProviderResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
resource "remnawave_infra_provider" "test" {
  name = "test-provider"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_infra_provider.test", "name", "test-provider"),
					resource.TestCheckResourceAttrSet("remnawave_infra_provider.test", "uuid"),
				),
			},
			{
				ResourceName:            "remnawave_infra_provider.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
				ImportStateIdFunc:       resourceUUIDImportStateID("remnawave_infra_provider.test"),
			},
		},
	})
}

func TestAccKeygenDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `data "remnawave_keygen" "current" {}`,
			Check:  resource.TestCheckResourceAttrSet("data.remnawave_keygen.current", "pub_key"),
		}},
	})
}
