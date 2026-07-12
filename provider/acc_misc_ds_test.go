package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccBandwidthRealtimeDataSource tests the remnawave_bandwidth_realtime data source.
func TestAccBandwidthRealtimeDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_bandwidth_realtime" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_bandwidth_realtime.test", "response"),
				),
			},
		},
	})
}

// TestAccSystemBandwidthStatsDataSource tests the remnawave_system_bandwidth_stats data source.
func TestAccSystemBandwidthStatsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_system_bandwidth_stats" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_system_bandwidth_stats.test", "response"),
				),
			},
		},
	})
}

// TestAccSystemNodesStatsDataSource tests the remnawave_system_nodes_stats data source.
func TestAccSystemNodesStatsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_system_nodes_stats" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_system_nodes_stats.test", "response"),
				),
			},
		},
	})
}

// TestAccSubscriptionRequestHistoryStatsDataSource tests the remnawave_subscription_request_history_stats data source.
func TestAccSubscriptionRequestHistoryStatsDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `data "remnawave_subscription_request_history_stats" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.remnawave_subscription_request_history_stats.test", "response"),
				),
			},
		},
	})
}

// TestAccConnectionKeysDataSource tests the remnawave_connection_keys data source.
func TestAccConnectionKeysDataSource(t *testing.T) {
	testAccPreCheck(t)

	endpoint, authBlock := testAccProviderBlock()
	providerCfg := fmt.Sprintf(testAccProviderConfig, endpoint, authBlock)

	// Uses a placeholder UUID; the test verifies the data source can be read.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerCfg + `
data "remnawave_connection_keys" "test" {
  uuid = "00000000-0000-0000-0000-000000000000"
}`,
				// The response depends on a valid subscription UUID existing in the panel.
				// We just verify the data source can be configured without schema errors.
			},
		},
	})
}
