# Terraform Provider for Remnawave

[![CI](https://github.com/batonogov/terraform-provider-remnawave/actions/workflows/ci.yml/badge.svg)](https://github.com/batonogov/terraform-provider-remnawave/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/batonogov/terraform-provider-remnawave)](go.mod)

A Terraform provider for [Remnawave](https://docs.rw) — a proxy management panel built on top of Xray-core. Manage VPN users, nodes, hosts, squads, billing, subscription pages, and more as infrastructure-as-code.

## Compatibility

**Support policy:** the provider supports Remnawave panel **v2.8.x**. The acceptance test suite runs against this version in CI on every push to `main` and every pull request.

| Remnawave version | Status |
| --- | --- |
| v2.8.0 | ✅ Tested |

*As Remnawave releases new versions, this table will be updated and the compatibility matrix expanded.*

## Features

### Resources (19)

| Resource | Description |
| --- | --- |
| `remnawave_user` | VPN user with traffic limits, expiration, VLESS/Trojan/Shadowsocks credentials |
| `remnawave_node` | Xray server node with traffic tracking, tags, consumption multipliers |
| `remnawave_host` | Connection endpoint (host) for VPN subscriptions with tags, nodes, mihomo |
| `remnawave_config_profile` | Xray config profile with inbounds, routing, sniffing |
| `remnawave_subscription_settings` | Subscription page settings (singleton) |
| `remnawave_external_squad` | External squad with templates, host overrides, HWID, custom remarks |
| `remnawave_internal_squad` | Internal squad with inbounds |
| `remnawave_subscription_template` | Subscription template (XRAY_JSON, MIHOMO, CLASH, SINGBOX, etc.) |
| `remnawave_panel_settings` | Panel branding, auth, passkey & OAuth2 settings (singleton) |
| `remnawave_snippet` | Xray config snippet (reusable JSON fragments) |
| `remnawave_node_plugin` | Node plugin (e.g. torrent blocker) |
| `remnawave_api_token` | API token with scopes |
| `remnawave_infra_provider` | Infrastructure billing provider |
| `remnawave_billing_node` | Infrastructure billing node (recurring billing) |
| `remnawave_billing_history` | Infrastructure billing history record (one-time payment) |
| `remnawave_subpage_config` | Subscription page config (i18n, theme, blocks — opaque JSON) |
| `remnawave_user_metadata` | Free-form key-value metadata for a user |
| `remnawave_node_metadata` | Free-form key-value metadata for a node |
| `remnawave_hwid_device` | HWID (Hardware ID) device entry for device-limit enforcement |

### Data Sources (20)

| Data Source | Description |
| --- | --- |
| `remnawave_nodes` | List all nodes with status and online user counts |
| `remnawave_users` | List all users with status and tags |
| `remnawave_hosts` | List all hosts |
| `remnawave_config_profiles` | List all config profiles |
| `remnawave_system_health` | Panel system health (raw JSON) |
| `remnawave_system_stats` | CPU, memory, uptime, user status counts, online stats |
| `remnawave_system_recap` | Monthly/total summary: users, traffic, nodes, version |
| `remnawave_system_bandwidth_stats` | System-level bandwidth statistics |
| `remnawave_system_nodes_stats` | System-level per-node statistics |
| `remnawave_nodes_metrics` | Per-node live metrics: users online, inbounds/outbounds stats |
| `remnawave_bandwidth_stats` | Per-node bandwidth usage by date range with sparkline data |
| `remnawave_bandwidth_stats_user` | Per-user bandwidth usage by date range |
| `remnawave_bandwidth_realtime` | Realtime bandwidth metrics per node |
| `remnawave_keygen` | Panel public key for node setup |
| `remnawave_subscriptions` | Fetch subscription by UUID/username/short UUID |
| `remnawave_subscription_request_history` | Subscription request history |
| `remnawave_subscription_request_history_stats` | Subscription request statistics |
| `remnawave_connection_keys` | Per-protocol connection keys for a user |
| `remnawave_hwid_stats` | HWID device statistics |
| `remnawave_hwid_top_users` | Top users by HWID device count |

## Quick Start

```hcl
terraform {
  required_providers {
    remnawave = {
      source  = "batonogov/remnawave"
      version = "~> 0.1"
    }
  }
}

provider "remnawave" {
  endpoint = "https://panel.example.com"
  username = "admin"
  password = var.remnawave_password
}

# Create a VPN user
resource "remnawave_user" "example" {
  username            = "john-doe"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240 # 10 GB
  description         = "Managed by Terraform"
}

# Create an external squad with a subscription template
resource "remnawave_subscription_template" "default" {
  name          = "vless-trojan"
  template_type = "XRAY_JSON"
}

resource "remnawave_external_squad" "default" {
  name = "Standard"
  templates = [
    { template_uuid = remnawave_subscription_template.default.uuid
      template_type = "XRAY_JSON" }
  ]
}

# Monitor system health
data "remnawave_system_stats" "current" {}
output "online_users" {
  value = data.remnawave_system_stats.current.online_now
}
```

## Authentication

The provider supports two authentication methods:

1. **API Token** (recommended) — generate one in the panel under *API Keys*. Set via `api_token` attribute or `REMNAWAVE_API_TOKEN` env var.
2. **Username/Password** — the provider logs in via `POST /api/auth/login` and obtains a JWT automatically. Auto-refreshes on 401.

## Environment Variables

| Variable | Description |
| --- | --- |
| `REMNAWAVE_ENDPOINT` | Panel URL |
| `REMNAWAVE_API_TOKEN` | API token (JWT) |
| `REMNAWAVE_USERNAME` | Admin username |
| `REMNAWAVE_PASSWORD` | Admin password |
| `REMNAWAVE_INSECURE_SKIP_VERIFY` | Skip TLS verification (`true`/`false`) |
| `REMNAWAVE_REQUEST_TIMEOUT` | HTTP timeout (default `30s`) |
| `REMNAWAVE_PROXY_HEADERS` | Send X-Forwarded-For/Proto headers (bypass ProxyCheckMiddleware) |

## Development

```bash
# Build
go build -o terraform-provider-remnawave

# Unit tests
go test ./provider -skip '^TestAcc' -count=1 -v

# Acceptance tests (requires Docker)
docker compose up -d --wait
# Register admin (first run only)
curl -sf -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"TestAdminPassword1234567"}'
# Run tests
TF_ACC=1 REMNAWAVE_ENDPOINT=http://localhost:3000 \
  REMNAWAVE_USERNAME=admin REMNAWAVE_PASSWORD=TestAdminPassword1234567 \
  go test ./provider -run TestAcc -count=1 -timeout 600s -v
```

## License

MIT
