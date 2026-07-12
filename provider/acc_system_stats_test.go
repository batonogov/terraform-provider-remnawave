package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSystemStatsDataSource tests the remnawave_system_stats data source.
func TestAccSystemStatsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_system_stats" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_system_stats.test", "cpu_cores"),
				),
			},
		},
	})
}

// TestAccSystemStatsDataSourceWithTZ tests the remnawave_system_stats data source with a timezone.
func TestAccSystemStatsDataSourceWithTZ(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_system_stats" "test" {
  tz = "Europe/Berlin"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_system_stats.test", "cpu_cores"),
					resource.TestCheckResourceAttr("data.remnawave_system_stats.test", "tz", "Europe/Berlin"),
				),
			},
		},
	})
}

// TestAccSystemRecapDataSource tests the remnawave_system_recap data source.
func TestAccSystemRecapDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_system_recap" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_system_recap.test", "version"),
				),
			},
		},
	})
}

// TestAccNodesMetricsDataSource tests the remnawave_nodes_metrics data source.
func TestAccNodesMetricsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_nodes_metrics" "test" {}`,
				// Nodes list may be empty on a fresh panel — just verify no error
			},
		},
	})
}
