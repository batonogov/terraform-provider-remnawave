# Terraform Provider for Remnawave

[![CI](https://github.com/batonogov/terraform-provider-remnawave/actions/workflows/ci.yml/badge.svg)](https://github.com/batonogov/terraform-provider-remnawave/actions/workflows/ci.yml)

A Terraform provider for [Remnawave](https://docs.rw) — a proxy management panel built on top of Xray-core. Manage VPN users, nodes, hosts and more as infrastructure-as-code.

## Features

### Resources

| Resource | Description |
| --- | --- |
| `remnawave_user` | VPN user with traffic limits, expiration, VLESS/Trojan/Shadowsocks credentials |
| `remnawave_node` | Xray server node with traffic tracking and config profile assignment |
| `remnawave_host` | Connection endpoint (host) for VPN subscriptions |

### Data Sources

| Data Source | Description |
| --- | --- |
| `remnawave_nodes` | List all nodes with status and online user counts |
| `remnawave_system_health` | Panel system health and statistics |

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

resource "remnawave_user" "example" {
  username            = "john-doe"
  expire_at           = "2027-01-01T00:00:00.000Z"
  traffic_limit_bytes = 10737418240 # 10 GB
  description         = "Managed by Terraform"
}
```

## Authentication

The provider supports two authentication methods:

1. **API Token** (recommended) — generate one in the panel under *API Keys*. Set via `api_token` attribute or `REMNAWAVE_API_TOKEN` env var.
2. **Username/Password** — the provider logs in via `POST /api/auth/login` and obtains a JWT automatically.

## Environment Variables

| Variable | Description |
| --- | --- |
| `REMNAWAVE_ENDPOINT` | Panel URL |
| `REMNAWAVE_API_TOKEN` | API token (JWT) |
| `REMNAWAVE_USERNAME` | Admin username |
| `REMNAWAVE_PASSWORD` | Admin password |
| `REMNAWAVE_INSECURE_SKIP_VERIFY` | Skip TLS verification (`true`/`false`) |
| `REMNAWAVE_REQUEST_TIMEOUT` | HTTP timeout (default `30s`) |

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
  -d '{"username":"admin","password":"TestAdminPassword123"}'
# Run tests
TF_ACC=1 REMNAWAVE_ENDPOINT=http://localhost:3000 \
  REMNAWAVE_USERNAME=admin REMNAWAVE_PASSWORD=TestAdminPassword123 \
  go test ./provider -run TestAcc -count=1 -timeout 600s -v
```

## License

MIT
