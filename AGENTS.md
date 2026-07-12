# AGENTS.md

This file provides guidance to AI coding agents when working with this repository.

## Project

Terraform provider for [Remnawave](https://docs.rw) — a proxy management panel
built on Xray-core. Go with `terraform-plugin-framework`.
Module: `github.com/batonogov/terraform-provider-remnawave`.
Registry: `batonogov/remnawave`. All provider code lives in `provider/`.

The Remnawave backend (`github.com/remnawave/backend`) is a NestJS TypeScript
application with a clean REST API. The panel uses PostgreSQL + Redis (Valkey).

**Compatibility:** Remnawave v2.8.x. Acceptance tests run against `:latest`
Docker image (pinned to v2.8.0).

## Commands

| Command | Description |
| --- | --- |
| `task build` | Build provider binary |
| `task fmt` | `gofmt -w provider/*.go` |
| `task vet` | `go vet ./...` |
| `task lint` | `golangci-lint run` |
| `task test:unit` | Unit tests (no Docker/Terraform needed) |
| `task test:acc` | Acceptance tests (Docker lifecycle included) |

## Architecture

### HTTP Client (`client.go`)

JWT Bearer auth. Two modes:
- **API token** (static JWT) — provided directly, no login needed
- **Username/password** — provider calls `POST /api/auth/login` to obtain JWT

Auto re-authenticates on 401 (unless using static API token).
All responses are wrapped in `{ "response": <data> }` envelope — `decodeResponse`
unwraps automatically.

`resolvePath` splits path and query string to avoid URL-encoding `?`.

### Resources (19)

| Resource | File | API Base |
| --- | --- | --- |
| `remnawave_user` | `resource_user.go` | `/api/users` |
| `remnawave_node` | `resource_node.go` | `/api/nodes` |
| `remnawave_host` | `resource_host.go` | `/api/hosts` |
| `remnawave_config_profile` | `resource_config_profile.go` | `/api/config-profiles` |
| `remnawave_subscription_settings` | `resource_subscription_settings.go` | `/api/subscription-settings` |
| `remnawave_external_squad` | `resource_external_squad.go` | `/api/external-squads` |
| `remnawave_internal_squad` | `resource_internal_squad.go` | `/api/internal-squads` |
| `remnawave_subscription_template` | `resource_subscription_template.go` | `/api/subscription-templates` |
| `remnawave_panel_settings` | `resource_panel_settings.go` | `/api/remnawave-settings` |
| `remnawave_snippet` | `resource_snippet.go` | `/api/snippets` |
| `remnawave_node_plugin` | `resource_node_plugin.go` | `/api/node-plugins` |
| `remnawave_api_token` | `resource_api_token.go` | `/api/tokens` |
| `remnawave_infra_provider` | `resource_infra_provider.go` | `/api/infra-billing/providers` |
| `remnawave_billing_node` | `resource_billing_node.go` | `/api/infra-billing/nodes` |
| `remnawave_billing_history` | `resource_billing_history.go` | `/api/infra-billing/history` |
| `remnawave_subpage_config` | `resource_subpage_config.go` | `/api/subscription-page-configs` |
| `remnawave_user_metadata` | `resource_user_metadata.go` | `/api/metadata/user/:uuid` |
| `remnawave_node_metadata` | `resource_node_metadata.go` | `/api/metadata/node/:uuid` |
| `remnawave_hwid_device` | `resource_hwid_device.go` | `/api/hwid/devices` |

### Data Sources (20)

Data sources live in `data_sources.go` (original) and `data_source_*.go` (newer).

| Data Source | File | API |
| --- | --- | --- |
| `remnawave_nodes` | `data_sources.go` | `/api/nodes` |
| `remnawave_users` | `data_sources.go` | `/api/users` |
| `remnawave_hosts` | `data_sources.go` | `/api/hosts` |
| `remnawave_config_profiles` | `data_sources.go` | `/api/config-profiles` |
| `remnawave_system_health` | `data_sources.go` | `/api/system/health` |
| `remnawave_keygen` | `data_sources.go` | `/api/keygen` |
| `remnawave_system_stats` | `data_source_system_stats.go` | `/api/system/stats` |
| `remnawave_system_recap` | `data_source_system_recap.go` | `/api/system/stats/recap` |
| `remnawave_nodes_metrics` | `data_source_nodes_metrics.go` | `/api/system/nodes/metrics` |
| `remnawave_bandwidth_stats` | `data_source_bandwidth.go` | `/api/bandwidth-stats/nodes` |
| `remnawave_bandwidth_stats_user` | `data_source_bandwidth.go` | `/api/bandwidth-stats/users/:uuid` |
| `remnawave_bandwidth_realtime` | `data_source_misc_stats.go` | `/api/bandwidth-stats/nodes/realtime` |
| `remnawave_system_bandwidth_stats` | `data_source_misc_stats.go` | `/api/system/stats/bandwidth` |
| `remnawave_system_nodes_stats` | `data_source_misc_stats.go` | `/api/system/stats/nodes` |
| `remnawave_subscriptions` | `data_source_subscriptions.go` | `/api/subscriptions/by-uuid/:uuid` |
| `remnawave_subscription_request_history` | `data_source_subscription_request_history.go` | `/api/subscription-request-history` |
| `remnawave_subscription_request_history_stats` | `data_source_misc_stats.go` | `/api/subscription-request-history/stats` |
| `remnawave_connection_keys` | `data_source_misc_stats.go` | `/api/subscriptions/connection-keys/:uuid` |
| `remnawave_hwid_stats` | `data_source_hwid.go` | `/api/hwid/devices/stats` |
| `remnawave_hwid_top_users` | `data_source_hwid.go` | `/api/hwid/devices/top-users` |

### Acceptance Tests

Acceptance tests run against a real Remnawave panel via Docker Compose
(`docker-compose.yaml`). The compose file spins up:
- `remnawave/backend` panel (port 3000)
- PostgreSQL 18
- Valkey (Redis) 9

Run with:
```bash
TF_ACC=1 REMNAWAVE_ENDPOINT=http://localhost:3000 \
  REMNAWAVE_USERNAME=admin REMNAWAVE_PASSWORD=admin \
  go test ./provider -run TestAcc -count=1 -timeout 600s -v
```

## Conventions

### Commits

Conventional Commits: `feat:`, `fix:`, `docs:`, `ci:`, `test:`, `chore:`.
Imperative mood, concise subjects.

### File naming

| Pattern | Example |
| --- | --- |
| Resources | `provider/resource_<name>.go` |
| Data sources | `provider/data_source_<name>.go` (or `data_sources.go` for small ones) |

### Testing

- Unit tests: `TestXxx` naming, table-driven where practical.
- Acceptance tests: `TestAccXxx`, `terraform-plugin-testing`,
  `ProtoV6ProviderFactories`.

### Auth

- API token auth preferred over username/password (avoids login on every plan).
- Panel env: `IS_DOCS_ENABLED=true` enables Swagger at `/docs` for API exploration.
- Panel env: `NODE_ENV=development` disables ProxyCheckMiddleware for direct access.
