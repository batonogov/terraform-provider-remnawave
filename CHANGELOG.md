# Changelog

## [0.3.0](https://github.com/batonogov/terraform-provider-remnawave/compare/v0.2.0...v0.3.0) (2026-07-16)


### Features

* add Remnawave 2.7 compatibility ([#78](https://github.com/batonogov/terraform-provider-remnawave/issues/78)) ([b002ab0](https://github.com/batonogov/terraform-provider-remnawave/commit/b002ab0f2dcd87c763870d039508484c1e788669))

## [0.2.0](https://github.com/batonogov/terraform-provider-remnawave/compare/v0.1.1...v0.2.0) (2026-07-13)


### Features

* expand Remnawave 2.8 API coverage ([#73](https://github.com/batonogov/terraform-provider-remnawave/issues/73)) ([dc91121](https://github.com/batonogov/terraform-provider-remnawave/commit/dc9112174db9b13aac08b6df50eebee98699e0bb))

## [0.1.1](https://github.com/batonogov/terraform-provider-remnawave/compare/v0.1.0...v0.1.1) (2026-07-13)


### Bug Fixes

* update release workflow for patch releases ([#66](https://github.com/batonogov/terraform-provider-remnawave/issues/66)) ([8667a6e](https://github.com/batonogov/terraform-provider-remnawave/commit/8667a6e44efad4cdc5d4fed800cc548c21fe1a9e))

## 0.1.0 (2026-07-12)


### Features

* add remnawave_users, remnawave_hosts, remnawave_config_profiles data sources ([#9](https://github.com/batonogov/terraform-provider-remnawave/issues/9)) ([63cda30](https://github.com/batonogov/terraform-provider-remnawave/commit/63cda307aaa2e0d026c6081faf335f00d4c7df14))
* bandwidth_realtime, system stats, connection_keys data sources ([#48](https://github.com/batonogov/terraform-provider-remnawave/issues/48)) ([852223b](https://github.com/batonogov/terraform-provider-remnawave/commit/852223b52a14afadaf9b078def5b8bb8eefc2ff5))
* bandwidth_stats and bandwidth_stats_user data sources ([#24](https://github.com/batonogov/terraform-provider-remnawave/issues/24), [#25](https://github.com/batonogov/terraform-provider-remnawave/issues/25)) ([20286fe](https://github.com/batonogov/terraform-provider-remnawave/commit/20286fea4e95bef1d3749d91079318fe009dd15d))
* billing_node and billing_history resources ([#19](https://github.com/batonogov/terraform-provider-remnawave/issues/19), [#20](https://github.com/batonogov/terraform-provider-remnawave/issues/20)) ([c86922d](https://github.com/batonogov/terraform-provider-remnawave/commit/c86922d6eb3a253e1f4fa941aa388dba7d328f30))
* extend external_squad with full fields ([#28](https://github.com/batonogov/terraform-provider-remnawave/issues/28)) ([#39](https://github.com/batonogov/terraform-provider-remnawave/issues/39)) ([bb70a82](https://github.com/batonogov/terraform-provider-remnawave/commit/bb70a828b2ae4dac71438980cd9022a4ce92e32c))
* extend node and host resources with missing fields ([#37](https://github.com/batonogov/terraform-provider-remnawave/issues/37)) ([86188be](https://github.com/batonogov/terraform-provider-remnawave/commit/86188bef331cc51a8df7198d37464c6cd3e9edab))
* extend panel_settings and subscription_template ([#31](https://github.com/batonogov/terraform-provider-remnawave/issues/31), [#32](https://github.com/batonogov/terraform-provider-remnawave/issues/32)) ([f6b37f4](https://github.com/batonogov/terraform-provider-remnawave/commit/f6b37f41aedbdfff4fbe5a932148859d7d5c70f9))
* hwid_device resource, hwid_stats and hwid_top_users data sources ([#42](https://github.com/batonogov/terraform-provider-remnawave/issues/42), [#43](https://github.com/batonogov/terraform-provider-remnawave/issues/43)) ([30ebbdc](https://github.com/batonogov/terraform-provider-remnawave/commit/30ebbdca173ff153361c6768e2716bb2302316f2))
* import support + internal_squad extension ([#62](https://github.com/batonogov/terraform-provider-remnawave/issues/62)) ([05e00f1](https://github.com/batonogov/terraform-provider-remnawave/commit/05e00f19800bee92170fb7865c0bbacf72edbdc6))
* initial terraform provider scaffold for Remnawave panel ([#1](https://github.com/batonogov/terraform-provider-remnawave/issues/1)) ([4402552](https://github.com/batonogov/terraform-provider-remnawave/commit/440255242016892d2bd721403cb92aaf82636683))
* Phase 2 — external_squad, internal_squad, subscription_template, panel_settings ([#15](https://github.com/batonogov/terraform-provider-remnawave/issues/15)) ([c180063](https://github.com/batonogov/terraform-provider-remnawave/commit/c180063c12285ea2401b5148013ad4bcdb746cf7))
* Phase 3+4 — snippet, node_plugin, api_token, infra_provider, keygen ([#16](https://github.com/batonogov/terraform-provider-remnawave/issues/16)) ([344c31e](https://github.com/batonogov/terraform-provider-remnawave/commit/344c31e33207d5f169dd2908b64c2143acb02242))
* Phase 5 — GoReleaser, release workflow, CONTRIBUTING, SECURITY, CODEOWNERS ([#17](https://github.com/batonogov/terraform-provider-remnawave/issues/17)) ([3ce6d06](https://github.com/batonogov/terraform-provider-remnawave/commit/3ce6d062d3d2240cc683d38933c606022f2e21f8)), closes [#7](https://github.com/batonogov/terraform-provider-remnawave/issues/7)
* remnawave_config_profile resource + data source ([#8](https://github.com/batonogov/terraform-provider-remnawave/issues/8)) ([5a5cbfc](https://github.com/batonogov/terraform-provider-remnawave/commit/5a5cbfc997d165927d87321bf142b26f26e4a7d7))
* remnawave_subpage_config resource ([#18](https://github.com/batonogov/terraform-provider-remnawave/issues/18)) ([f29a7a8](https://github.com/batonogov/terraform-provider-remnawave/commit/f29a7a8ea84b936f83bb6120c8efb7e97d46b01d))
* remnawave_subscription_settings resource + fix GetAllUsers ([#11](https://github.com/batonogov/terraform-provider-remnawave/issues/11)) ([29e95f0](https://github.com/batonogov/terraform-provider-remnawave/commit/29e95f03de164fdc7027e23af4ecf3100260a652))
* subscriptions and subscription_request_history data sources ([#26](https://github.com/batonogov/terraform-provider-remnawave/issues/26), [#27](https://github.com/batonogov/terraform-provider-remnawave/issues/27)) ([#36](https://github.com/batonogov/terraform-provider-remnawave/issues/36)) ([477d313](https://github.com/batonogov/terraform-provider-remnawave/commit/477d31356d392da8038f696ea0755ea64ceb75f2))
* system_stats, system_recap, nodes_metrics data sources ([#21](https://github.com/batonogov/terraform-provider-remnawave/issues/21), [#22](https://github.com/batonogov/terraform-provider-remnawave/issues/22), [#23](https://github.com/batonogov/terraform-provider-remnawave/issues/23)) ([9d2d0eb](https://github.com/batonogov/terraform-provider-remnawave/commit/9d2d0ebd34e69b71f0e164d10c0455b20c2d50fd))
* user_metadata and node_metadata resources ([#41](https://github.com/batonogov/terraform-provider-remnawave/issues/41)) ([fef475f](https://github.com/batonogov/terraform-provider-remnawave/commit/fef475fc59bbbfdfe1fde1b2de4b8ae3d1f616e8))


### Bug Fixes

* config_profile default config + subscription_settings UUID + add tests ([#13](https://github.com/batonogov/terraform-provider-remnawave/issues/13)) ([8e7b715](https://github.com/batonogov/terraform-provider-remnawave/commit/8e7b715efaf7afdf724c7ab9cc4082574e25ac79))
* force release-as 0.1.0 for first release ([ebdfd59](https://github.com/batonogov/terraform-provider-remnawave/commit/ebdfd59fc9bf7f7cbc4a2bde7b62b06023d82a98))
* mark config as Computed and populate from API on create ([49bb778](https://github.com/batonogov/terraform-provider-remnawave/commit/49bb7789717cfe6c3e558426b4b416728b86edd5))
* pin release-please to v0.1.0 (not v1.0.0) ([28abc91](https://github.com/batonogov/terraform-provider-remnawave/commit/28abc91853489589dc059d87e0405a2084ddd358))
* resolvePath URL-encodes query string — split path and query ([16824b5](https://github.com/batonogov/terraform-provider-remnawave/commit/16824b5089891dee94bb3ffd2d8f868c48869f72))
* resource import via SetAttribute for UUID ([#10](https://github.com/batonogov/terraform-provider-remnawave/issues/10)) ([78e43f5](https://github.com/batonogov/terraform-provider-remnawave/commit/78e43f5e3acd716a7b496591d02032a4a2966b5d))
* restore subscription + external_squad types lost in merge ([249fc68](https://github.com/batonogov/terraform-provider-remnawave/commit/249fc68e88ff84abf1b97747e976878d8e83ecb5))
* send null for nullable fields required by billing node API ([f805cc3](https://github.com/batonogov/terraform-provider-remnawave/commit/f805cc317d4af84b52e42d301d4379e645bd4244))

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
