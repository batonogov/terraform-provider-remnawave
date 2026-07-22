# Changelog

## [0.6.0](https://github.com/batonogov/terraform-provider-remnawave/compare/v0.5.1...v0.6.0) (2026-07-22)


### Features

* support custom HTTP headers ([#156](https://github.com/batonogov/terraform-provider-remnawave/issues/156)) ([250bf54](https://github.com/batonogov/terraform-provider-remnawave/commit/250bf5473f6285071f9c5884e7c26075c2923b18))


### Bug Fixes

* align bulk actions and host tags across versions ([#151](https://github.com/batonogov/terraform-provider-remnawave/issues/151)) ([0aa8beb](https://github.com/batonogov/terraform-provider-remnawave/commit/0aa8bebce20dc1b4396718d89822a6c6fc52bd7e))

## [0.5.1](https://github.com/batonogov/terraform-provider-remnawave/compare/v0.5.0...v0.5.1) (2026-07-20)


### Bug Fixes

* correct fetch-ips-result endpoint path and response schema ([#141](https://github.com/batonogov/terraform-provider-remnawave/issues/141)) ([#144](https://github.com/batonogov/terraform-provider-remnawave/issues/144)) ([3b93881](https://github.com/batonogov/terraform-provider-remnawave/commit/3b93881c9263de9aa1536884cfb4001a85b3f10f))
* unwrap passkeys response envelope ([#145](https://github.com/batonogov/terraform-provider-remnawave/issues/145)) ([aa770aa](https://github.com/batonogov/terraform-provider-remnawave/commit/aa770aa0bd616f91ff84362945e9ee39ef0bb0f1))

## [0.5.0](https://github.com/batonogov/terraform-provider-remnawave/compare/v0.4.1...v0.5.0) (2026-07-19)


### Features

* add enum and format validators for all constrained fields ([#111](https://github.com/batonogov/terraform-provider-remnawave/issues/111)) ([#131](https://github.com/batonogov/terraform-provider-remnawave/issues/131)) ([7b74a1c](https://github.com/batonogov/terraform-provider-remnawave/commit/7b74a1cecdce4a30933f2f7f8b3b3ade09a2211c))
* add OAuth2 settings to panel_settings ([#112](https://github.com/batonogov/terraform-provider-remnawave/issues/112)) ([#132](https://github.com/batonogov/terraform-provider-remnawave/issues/132)) ([80f4b16](https://github.com/batonogov/terraform-provider-remnawave/commit/80f4b1630cb2dd707c4e8188f3aa0591906fc637))
* add user and node bulk action resources ([#114](https://github.com/batonogov/terraform-provider-remnawave/issues/114)) ([#133](https://github.com/batonogov/terraform-provider-remnawave/issues/133)) ([8d35d16](https://github.com/batonogov/terraform-provider-remnawave/commit/8d35d1613ad1545d7e96ea2b86edaee90ad6cffd))
* extend drop_connections to full API schema ([#113](https://github.com/batonogov/terraform-provider-remnawave/issues/113)) ([#130](https://github.com/batonogov/terraform-provider-remnawave/issues/130)) ([65645c0](https://github.com/batonogov/terraform-provider-remnawave/commit/65645c0b008151b99b6450ece41301963d7ef790))


### Bug Fixes

* unwrap host tags response envelope ([#136](https://github.com/batonogov/terraform-provider-remnawave/issues/136)) ([#137](https://github.com/batonogov/terraform-provider-remnawave/issues/137)) ([24714e5](https://github.com/batonogov/terraform-provider-remnawave/commit/24714e5108404a8590214f3850143612042ac02f))

## [0.4.1](https://github.com/batonogov/terraform-provider-remnawave/compare/v0.4.0...v0.4.1) (2026-07-18)


### Bug Fixes

* add RequiresReplace for billing_node immutable fields ([#107](https://github.com/batonogov/terraform-provider-remnawave/issues/107)) ([#123](https://github.com/batonogov/terraform-provider-remnawave/issues/123)) ([7894977](https://github.com/batonogov/terraform-provider-remnawave/commit/78949776d50496421f4517a766216999d8d1fda8))
* allow clearing external_squad_uuid and active_internal_squads on user update ([#108](https://github.com/batonogov/terraform-provider-remnawave/issues/108)) ([#126](https://github.com/batonogov/terraform-provider-remnawave/issues/126)) ([e17acb8](https://github.com/batonogov/terraform-provider-remnawave/commit/e17acb84386726644e038c2d84b4a3e2334e98aa))
* pass size=1000 to paginated endpoints ([#106](https://github.com/batonogov/terraform-provider-remnawave/issues/106)) ([#122](https://github.com/batonogov/terraform-provider-remnawave/issues/122)) ([5dd41b1](https://github.com/batonogov/terraform-provider-remnawave/commit/5dd41b1eec8ccce418300357252157f2808cb590))
* strip credential fields from user PATCH payload ([#110](https://github.com/batonogov/terraform-provider-remnawave/issues/110)) ([#124](https://github.com/batonogov/terraform-provider-remnawave/issues/124)) ([beb7f15](https://github.com/batonogov/terraform-provider-remnawave/commit/beb7f152bab13b048a09f8bc2bfe5a85213d0efe))
* unify action naming to underscore, keep hyphen as backward-compatible alias ([#115](https://github.com/batonogov/terraform-provider-remnawave/issues/115)) ([#127](https://github.com/batonogov/terraform-provider-remnawave/issues/127)) ([256f324](https://github.com/batonogov/terraform-provider-remnawave/commit/256f3247bb089d2884ce7dcbc0c26ea28b1e6b61))
* use API response in infra_provider and internal_squad Update ([#109](https://github.com/batonogov/terraform-provider-remnawave/issues/109)) ([#121](https://github.com/batonogov/terraform-provider-remnawave/issues/121)) ([4bf4f7c](https://github.com/batonogov/terraform-provider-remnawave/commit/4bf4f7ce98a5c9b306a4e0ff7a5771df76fe5608))

## [0.4.0](https://github.com/batonogov/terraform-provider-remnawave/compare/v0.3.0...v0.4.0) (2026-07-17)


### Features

* add branding_logo_url and password_auth_enabled to panel_settings ([#97](https://github.com/batonogov/terraform-provider-remnawave/issues/97)) ([6144aaf](https://github.com/batonogov/terraform-provider-remnawave/commit/6144aaf59d306369c0d9d2ce191aca91891dafa1))
* add host tags data source and host bulk action resource ([#104](https://github.com/batonogov/terraform-provider-remnawave/issues/104)) ([6d8af2f](https://github.com/batonogov/terraform-provider-remnawave/commit/6d8af2fcc4c8f995ada08cde5e358ed72ef03e4d))
* add IP Control data sources and drop_connections resource ([#102](https://github.com/batonogov/terraform-provider-remnawave/issues/102)) ([a21e2d0](https://github.com/batonogov/terraform-provider-remnawave/commit/a21e2d03ee8191c6f6dcd8686306e167f75f408c))
* add remnawave_node_action resource ([#103](https://github.com/batonogov/terraform-provider-remnawave/issues/103)) ([1aeebc0](https://github.com/batonogov/terraform-provider-remnawave/commit/1aeebc00bea936e64b63f78f3bc4a3703050d8d2))
* add remnawave_passkeys data source and passkey resource ([#105](https://github.com/batonogov/terraform-provider-remnawave/issues/105)) ([c6d2789](https://github.com/batonogov/terraform-provider-remnawave/commit/c6d2789b3e69d49d2155ee4c6438692c4c5d8928))
* add remnawave_user_action resource ([#101](https://github.com/batonogov/terraform-provider-remnawave/issues/101)) ([2808783](https://github.com/batonogov/terraform-provider-remnawave/commit/2808783a64f2d4e2d47c7f209177e2339c36957a))


### Bug Fixes

* **acc:** skip FullNodeStack on 2.7.x (host tags provider bug) ([#96](https://github.com/batonogov/terraform-provider-remnawave/issues/96)) ([facdc97](https://github.com/batonogov/terraform-provider-remnawave/commit/facdc9753a1c40392dd1fb4779e8a8025addad21))

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
