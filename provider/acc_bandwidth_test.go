package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBandwidthStatsDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
data "remnawave_bandwidth_stats" "test" {
  start = "2025-01-01"
  end   = "2025-12-31"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.remnawave_bandwidth_stats.test", "categories.#"),
			),
		}},
	})
}

func TestAccBandwidthStatsUserDataSource(t *testing.T) {
	testAccPreCheck(t)
	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerCfg + `
resource "remnawave_user" "test" {
  username            = "bw-test-user"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240
}

data "remnawave_bandwidth_stats_user" "test" {
  uuid  = remnawave_user.test.uuid
  start = "2025-01-01"
  end   = "2025-12-31"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.remnawave_bandwidth_stats_user.test", "categories.#"),
			),
		}},
	})
}
