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
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_snippet" "test" {
  name    = "test-snippet-2"
  snippet = jsonencode([{ "type" = "field", "domain" = ["geosite:category-ads"] }])
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("remnawave_snippet.test", "name", "test-snippet-2"),
			),
		}},
	})
}

func TestAccNodePluginResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_node_plugin" "test" {
  name = "test-plugin"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("remnawave_node_plugin.test", "name", "test-plugin"),
				resource.TestCheckResourceAttrSet("remnawave_node_plugin.test", "uuid"),
			),
		}},
	})
}

func TestAccApiTokenResource(t *testing.T) {
	// API token CRUD requires admin JWT, not API token auth
	testAccPreCheck(t)
	if os.Getenv(envAPIToken) != "" {
		t.Skip("api_token resource requires admin JWT — skipped when using api_token auth")
	}
}

func TestAccInfraProviderResource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_infra_provider" "test" {
  name = "test-provider"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("remnawave_infra_provider.test", "name", "test-provider"),
				resource.TestCheckResourceAttrSet("remnawave_infra_provider.test", "uuid"),
			),
		}},
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
