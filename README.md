<!-- markdownlint-disable first-line-h1 no-inline-html -->

<div align="center">

# Terraform Provider for Remnawave

[![CI](https://github.com/batonogov/terraform-provider-remnawave/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/batonogov/terraform-provider-remnawave/actions/workflows/ci.yml)
[![Terraform Registry](https://img.shields.io/badge/Registry-batonogov%2Fremnawave-844FBA?logo=terraform&logoColor=white)](https://registry.terraform.io/providers/batonogov/remnawave/latest)
[![Latest Release](https://img.shields.io/github/v/release/batonogov/terraform-provider-remnawave?color=blue)](https://github.com/batonogov/terraform-provider-remnawave/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/batonogov/terraform-provider-remnawave?logo=go&logoColor=white)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Context7](https://img.shields.io/badge/Context7-Indexed-blue)](https://context7.com/batonogov/terraform-provider-remnawave)

</div>

A community Terraform provider for [**Remnawave**](https://docs.rw) — a modern
proxy management panel built on top of Xray-core. Manage VPN users, nodes,
hosts, squads, billing, subscription pages, and more as infrastructure-as-code.

---

## Table of Contents

- [Why Terraform + Remnawave?](#why-terraform--remnawave)
- [Compatibility](#compatibility)
- [Quick Start](#quick-start)
- [Authentication](#authentication)
- [Provider Configuration](#provider-configuration)
- [Resources](#resources)
- [Data Sources](#data-sources)
- [Examples](#examples)
- [Importing Existing Resources](#importing-existing-resources)
- [Versioning \& Upgrades](#versioning--upgrades)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Why Terraform + Remnawave?

| Without Terraform | With this provider |
| --- | --- |
| Create users one-by-one in the web UI | Define hundreds of users in HCL, `terraform apply` |
| Manual config drift between nodes | Version-controlled, reproducible node configs |
| No audit trail for changes | Git history + Terraform plan/apply logs |
| Onboarding = manual clicks | Onboarding = `terraform init && terraform apply` |

## Compatibility

The provider supports Remnawave panel **v2.7.x and v2.8.x**. The acceptance test
suite runs against both versions in CI on every push to `main` and every pull
request — [see the full CI matrix](https://github.com/batonogov/terraform-provider-remnawave/actions/workflows/ci.yml).

| Remnawave version | Status |
| --- | --- |
| v2.8.0 | ✅ Tested (primary) |
| v2.7.4 | ✅ Tested (matrix) |

The client auto-detects the backend version via `/api/system/metadata` on the
first API-token operation and adapts API calls where contracts differ between
versions. No configuration is required.

## Quick Start

```hcl
terraform {
  required_providers {
    remnawave = {
      source  = "batonogov/remnawave"
      version = "~> 0.5.0"
    }
  }
}

provider "remnawave" {
  endpoint  = "https://panel.example.com"
  api_token = var.remnawave_api_token
}

# ─── VPN user with 10 GB traffic limit ───
resource "remnawave_user" "john" {
  username            = "john-doe"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240 # 10 GB
  traffic_limit_strategy = "MONTH"
  description         = "Managed by Terraform"
}

# ─── Xray node ───
resource "remnawave_config_profile" "default" {
  name   = "default-profile"
  config = file("xray-config.json")
}

resource "remnawave_node" "de-fra-01" {
  name                 = "de-fra-01"
  address              = "1.2.3.4"
  port                 = 443
  config_profile_uuid  = remnawave_config_profile.default.uuid
  config_profile_inbounds = [remnawave_config_profile.default.inbounds[0].uuid]
}

# ─── Connection host ───
resource "remnawave_host" "de-fra-01-vless" {
  remark                      = "🇩🇪 Frankfurt"
  address                     = "vpn.example.com"
  port                        = 443
  config_profile_uuid         = remnawave_config_profile.default.uuid
  config_profile_inbound_uuid = remnawave_config_profile.default.inbounds[0].uuid
}

# ─── Monitor system health ───
data "remnawave_system_stats" "current" {}

output "online_users" {
  value = data.remnawave_system_stats.current.online_now
}
```

## Authentication

The provider supports two authentication methods:

### API Token (recommended)

Generate a token in the panel under **API Keys** and pass it via the `api_token`
attribute or the `REMNAWAVE_API_TOKEN` environment variable. This is a static
JWT — no login round-trip, works with scoped permissions.

```hcl
provider "remnawave" {
  endpoint  = "https://panel.example.com"
  api_token = var.remnawave_api_token
}
```

### Username / Password

If no API token is provided, the provider logs in via `POST /api/auth/login`
and obtains a JWT automatically. The token is refreshed on 401 responses.

```hcl
provider "remnawave" {
  endpoint = "https://panel.example.com"
  username = "admin"
  password = var.remnawave_password
}
```

> **Note:** Username/password auth requires setting `proxy_headers = true` or
> `NODE_ENV=development` on the backend — Remnawave's `ProxyCheckMiddleware` requires
> `X-Forwarded-For`/`X-Forwarded-Proto` headers for browser-originated requests.

## Provider Configuration

All provider attributes can be supplied through environment variables — the
provider block may be empty when the environment is configured. Explicit HCL
values take precedence over environment variables.

| Attribute | Env var | Type | Description |
| --- | --- | --- | --- |
| `endpoint` | `REMNAWAVE_ENDPOINT` | `string` | Base URL of the panel, e.g. `https://panel.example.com` |
| `api_token` | `REMNAWAVE_API_TOKEN` | `string` (sensitive) | Pre-generated API token (JWT). If set, `username`/`password` are ignored |
| `username` | `REMNAWAVE_USERNAME` | `string` | Admin username for login |
| `password` | `REMNAWAVE_PASSWORD` | `string` (sensitive) | Admin password for login |
| `insecure_skip_verify` | `REMNAWAVE_INSECURE_SKIP_VERIFY` | `bool` | Skip TLS certificate verification (`true`/`false`) |
| `request_timeout` | `REMNAWAVE_REQUEST_TIMEOUT` | `string` | HTTP client timeout (default `30s`) |
| `proxy_headers` | `REMNAWAVE_PROXY_HEADERS` | `bool` | Send `X-Forwarded-For`/`Proto` headers (bypass `ProxyCheckMiddleware`) |

## Resources

The provider exposes **26 resources** across 7 functional areas:

### Core VPN Management

| Resource | Description |
| --- | --- |
| [`remnawave_user`](docs/resources/user.md) | VPN user with traffic limits, expiration, VLESS/Trojan/Shadowsocks credentials |
| [`remnawave_node`](docs/resources/node.md) | Xray server node with traffic tracking, tags, consumption multipliers |
| [`remnawave_host`](docs/resources/host.md) | Connection endpoint (host) for VPN subscriptions |
| [`remnawave_config_profile`](docs/resources/config_profile.md) | Xray config profile with inbounds, routing, sniffing |

### Access Control & Routing

| Resource | Description |
| --- | --- |
| [`remnawave_external_squad`](docs/resources/external_squad.md) | External squad (managed by name) |
| [`remnawave_internal_squad`](docs/resources/internal_squad.md) | Internal squad with inbounds and accessible nodes |
| [`remnawave_subscription_template`](docs/resources/subscription_template.md) | Subscription template (XRAY_JSON, MIHOMO, CLASH, SINGBOX, etc.) |
| [`remnawave_subscription_settings`](docs/resources/subscription_settings.md) | Subscription page settings (singleton) |
| [`remnawave_subpage_config`](docs/resources/subpage_config.md) | Subscription page config (i18n, theme, blocks) |

### Panel & Branding

| Resource | Description |
| --- | --- |
| [`remnawave_panel_settings`](docs/resources/panel_settings.md) | Panel branding, auth, passkey settings (singleton) |
| [`remnawave_snippet`](docs/resources/snippet.md) | Xray config snippet (reusable JSON fragments) |
| [`remnawave_node_plugin`](docs/resources/node_plugin.md) | Node plugin (e.g. torrent blocker) |

### Infrastructure Billing

| Resource | Description |
| --- | --- |
| [`remnawave_infra_provider`](docs/resources/infra_provider.md) | Infrastructure billing provider |
| [`remnawave_billing_node`](docs/resources/billing_node.md) | Infrastructure billing node (recurring billing) |
| [`remnawave_billing_history`](docs/resources/billing_history.md) | Infrastructure billing history record (one-time payment) |

### API & Access Tokens

| Resource | Description |
| --- | --- |
| [`remnawave_api_token`](docs/resources/api_token.md) | API token with scopes and expiration |
| [`remnawave_passkey`](docs/resources/passkey.md) | WebAuthn passkey (import-only — cannot be created via Terraform) |

### Metadata & Devices

| Resource | Description |
| --- | --- |
| [`remnawave_user_metadata`](docs/resources/user_metadata.md) | Free-form key-value metadata for a user |
| [`remnawave_node_metadata`](docs/resources/node_metadata.md) | Free-form key-value metadata for a node |
| [`remnawave_hwid_device`](docs/resources/hwid_device.md) | HWID device entry for device-limit enforcement |

### Imperative Actions

These resources trigger one-shot operations on `terraform apply`. Use the
`triggers` attribute to force re-execution when external values change.

| Resource | Description |
| --- | --- |
| [`remnawave_node_action`](docs/resources/node_action.md) | Enable / disable / restart / reset traffic on a node |
| [`remnawave_user_action`](docs/resources/user_action.md) | Enable / disable / reset traffic / revoke subscription on a user |
| [`remnawave_host_bulk_action`](docs/resources/host_bulk_action.md) | Bulk enable / disable / delete hosts |
| [`remnawave_user_bulk_action`](docs/resources/user_bulk_action.md) | Bulk reset traffic / revoke subscriptions / delete users / extend expiration |
| [`remnawave_node_bulk_action`](docs/resources/node_bulk_action.md) | Bulk enable / disable / restart / reset traffic on nodes |
| [`remnawave_drop_connections`](docs/resources/drop_connections.md) | Drop active connections by user UUID or IP address on all or selected nodes |

## Data Sources

**23 data sources** for reading panel state:

### Inventory

| Data Source | Description |
| --- | --- |
| [`remnawave_nodes`](docs/data-sources/nodes.md) | List all nodes with status and online user counts |
| [`remnawave_users`](docs/data-sources/users.md) | List all users with status and tags |
| [`remnawave_hosts`](docs/data-sources/hosts.md) | List all hosts |
| [`remnawave_config_profiles`](docs/data-sources/config_profiles.md) | List all config profiles |
| [`remnawave_host_tags`](docs/data-sources/host_tags.md) | List all unique host tags |
| [`remnawave_passkeys`](docs/data-sources/passkeys.md) | List WebAuthn passkeys for the current admin |

### System & Health

| Data Source | Description |
| --- | --- |
| [`remnawave_system_health`](docs/data-sources/system_health.md) | Panel system health (raw JSON) |
| [`remnawave_system_stats`](docs/data-sources/system_stats.md) | CPU, memory, uptime, user status counts, online stats |
| [`remnawave_system_recap`](docs/data-sources/system_recap.md) | Monthly/total summary: users, traffic, nodes, version |
| [`remnawave_system_bandwidth_stats`](docs/data-sources/system_bandwidth_stats.md) | System-level bandwidth statistics |
| [`remnawave_system_nodes_stats`](docs/data-sources/system_nodes_stats.md) | System-level per-node statistics |

### Bandwidth & Metrics

| Data Source | Description |
| --- | --- |
| [`remnawave_nodes_metrics`](docs/data-sources/nodes_metrics.md) | Per-node live metrics: users online, inbounds/outbounds |
| [`remnawave_bandwidth_stats`](docs/data-sources/bandwidth_stats.md) | Per-node bandwidth usage by date range |
| [`remnawave_bandwidth_stats_user`](docs/data-sources/bandwidth_stats_user.md) | Per-user bandwidth usage by date range |
| [`remnawave_bandwidth_realtime`](docs/data-sources/bandwidth_realtime.md) | Realtime bandwidth metrics per node |

### Subscriptions

| Data Source | Description |
| --- | --- |
| [`remnawave_subscriptions`](docs/data-sources/subscriptions.md) | Fetch subscription by UUID/username/short UUID |
| [`remnawave_subscription_request_history`](docs/data-sources/subscription_request_history.md) | Subscription request history |
| [`remnawave_subscription_request_history_stats`](docs/data-sources/subscription_request_history_stats.md) | Subscription request statistics |
| [`remnawave_connection_keys`](docs/data-sources/connection_keys.md) | Per-protocol connection keys for a user |

### Other

| Data Source | Description |
| --- | --- |
| [`remnawave_keygen`](docs/data-sources/keygen.md) | Panel public key for node setup |
| [`remnawave_user_ips`](docs/data-sources/user_ips.md) | Fetch IPs a user is currently connected from |
| [`remnawave_hwid_stats`](docs/data-sources/hwid_stats.md) | HWID device statistics |
| [`remnawave_hwid_top_users`](docs/data-sources/hwid_top_users.md) | Top users by HWID device count |

## Examples

Browse the [`examples/`](examples/) directory for focused configuration
examples. Some snippets reference resources or variables supplied by the
surrounding configuration:

```bash
examples/
├── resources/
│   ├── remnawave_user/            # User with traffic limit + tag
│   ├── remnawave_node_action/     # Periodic traffic reset via triggers
│   ├── remnawave_host_bulk_action/# Bulk enable/disable/delete
│   ├── remnawave_user_bulk_action/# Bulk user operations
│   ├── remnawave_node_bulk_action/# Bulk node operations
│   ├── remnawave_user_action/     # Enable/disable/reset/revoke
│   └── remnawave_passkey/         # Import + manage existing passkeys
└── data-sources/
    ├── remnawave_users/           # List all users
    ├── remnawave_system_stats/    # Dashboard metrics
    └── remnawave_passkeys/        # List passkeys
```

Full documentation for every resource and data source is on the
[Terraform Registry](https://registry.terraform.io/providers/batonogov/remnawave/latest/docs)
and in [`docs/`](docs/).

## Importing Existing Resources

Resources created outside Terraform (e.g. via the web UI) can be imported into
Terraform state:

```bash
# Import a user by UUID
terraform import remnawave_user.john 550e8400-e29b-41d4-a716-446655440000

# Import a node by UUID
terraform import remnawave_node.de-fra-01 550e8400-e29b-41d4-a716-446655440001

# Import a passkey (import-only resource)
terraform import remnawave_passkey.admin 550e8400-e29b-41d4-a716-446655440002
```

Import IDs are resource-specific. Most stateful resources use a UUID, snippets
use their name, and resources with compound identities document their required
format on the corresponding resource page.

## Versioning & Upgrades

This provider follows [Semantic Versioning](https://semver.org/). Breaking
changes are reserved for major releases (e.g. `1.0.0`).

### Upgrading

```bash
# Upgrade to the latest version
terraform init -upgrade
```

Review the [CHANGELOG](CHANGELOG.md) for breaking changes and migration notes
before upgrading across major versions.

### Version Constraints

```hcl
terraform {
  required_providers {
    remnawave = {
      source  = "batonogov/remnawave"
      version = "~> 0.5.0" # Allow 0.5.x patch releases
    }
  }
}
```

## Development

### Requirements

| Tool | Version |
| --- | --- |
| [Go](https://go.dev/dl/) | 1.26.4+ (see [`go.mod`](go.mod)) |
| [Terraform CLI](https://developer.hashicorp.com/terraform/install) | 1.12+ |
| [Task](https://taskfile.dev) | Latest (optional, for `task` commands) |
| [Docker](https://www.docker.com/) | Required for acceptance tests |

### Build & Test

```bash
# Build the provider binary
go build -o terraform-provider-remnawave

# Format code
gofmt -w provider/*.go

# Lint
golangci-lint run

# Unit tests (no Docker needed)
go test ./provider/... -race -cover

# Acceptance tests (starts Remnawave panel via Docker Compose)
task test:acc
```

See [`Taskfile.yml`](Taskfile.yml) and [`AGENTS.md`](AGENTS.md) for the full
list of commands and development conventions.

### Acceptance Tests

Acceptance tests run against a real Remnawave panel via Docker Compose
(`docker-compose.yaml`). The compose file spins up:

- `remnawave/backend` panel (port 3000)
- PostgreSQL 18
- Valkey (Redis) 9

```bash
# Run the complete Docker lifecycle + test suite
task test:acc

# Override the Remnawave version under test
REMNAWAVE_VERSION=2.9.0 REMNAWAVE_DIGEST=sha256:<digest> task test:acc
```

All compose images are pinned by `sha256` digest for reproducibility.

### Regenerating Documentation

Documentation is auto-generated from provider schemas via
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs):

```bash
task docs       # Format examples + regenerate Registry docs
task docs:check # Validate and detect stale generated docs (CI gate)
```

## Contributing

Contributions are welcome! Please:

1. Open an [issue](https://github.com/batonogov/terraform-provider-remnawave/issues)
   to discuss the change before starting work.
2. Follow [Conventional Commits](https://www.conventionalcommits.org/) for
   commit messages (`feat:`, `fix:`, `docs:`, `test:`, `chore:`).
3. Add or update tests for any changed behavior.
4. Run `gofmt`, `golangci-lint`, and the relevant test suite before submitting
   a pull request.
5. Ensure CI is green — acceptance tests run against both Remnawave v2.7.4 and
   v2.8.0.

### Security

If you believe you have found a security issue, please report it privately
rather than opening a public issue.

## License

[MIT](LICENSE) © 2026 Fedor Batonogov
