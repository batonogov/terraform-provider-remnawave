data "remnawave_node_integrations" "all" {}

output "node_integration_ids" {
  value = {
    for integration in data.remnawave_node_integrations.all.node_integrations :
    integration.name => integration.uuid
  }
}
