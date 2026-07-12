# Changelog

## v0.1.0

### Features

* Initial release of the Terraform Provider for Remnawave.
* **19 resources:** user, node, host, config_profile, subscription_settings,
  external_squad, internal_squad, subscription_template, panel_settings,
  snippet, node_plugin, api_token, infra_provider, billing_node,
  billing_history, subpage_config, user_metadata, node_metadata, hwid_device.
* **20 data sources:** nodes, users, hosts, config_profiles, system_health,
  system_stats, system_recap, nodes_metrics, bandwidth_stats,
  bandwidth_stats_user, bandwidth_realtime, system_bandwidth_stats,
  system_nodes_stats, keygen, subscriptions, subscription_request_history,
  subscription_request_history_stats, connection_keys, hwid_stats,
  hwid_top_users.
* JWT authentication (API token + username/password).
* Acceptance test suite with Docker Compose panel.
* CI: lint, build, unit tests, acceptance tests.
* GoReleaser config for cross-platform binary builds.
* Compatibility: Remnawave v2.8.x.
