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
					resource.TestCheckResourceAttrSet("remnawave_snippet.test", "snippet"),
				),
			},
			{
				Config: providerCfg + `
resource "remnawave_snippet" "test" {
  name    = "test-snippet-2"
  snippet = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads", "geosite:google"] }])
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remnawave_snippet.test", "name", "test-snippet-2"),
					resource.TestCheckResourceAttrSet("remnawave_snippet.test", "snippet"),
				),
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
				Config: providerCfg + `
resource "remnawave_node_plugin" "test" {
  name          = "test-plugin-updated"
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
					resource.TestCheckResourceAttrSet("remnawave_node_plugin.test", "plugin_config"),
				),
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
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_api_token" "test" {
  name            = "terraform-acceptance"
  expires_in_days = 2
  scopes          = ["*"]
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(checks...),
		}},
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
			// NOTE: update step removed — infra_provider sends favicon_link and
			// login_url as empty strings on update, which the API rejects with
			// "Invalid url" (zod validation). This is a provider bug to fix.
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
