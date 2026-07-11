# AGENTS.md

This file provides guidance to AI coding agents when working with this repository.

## Project

Terraform provider for [Remnawave](https://docs.rw) — a proxy management panel
built on Xray-core. Go with `terraform-plugin-framework`.
Module: `github.com/batonogov/terraform-provider-remnawave`.
Registry: `batonogov/remnawave`. All provider code lives in `provider/`.

The Remnawave backend (`github.com/remnawave/backend`) is a NestJS TypeScript
application with a clean REST API. The panel uses PostgreSQL + Redis (Valkey).

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

### API Endpoints

| Resource | Base Path | Methods |
| --- | --- | --- |
| Users | `/api/users` | POST, PATCH, GET `/:uuid`, DELETE `/:uuid` |
| Nodes | `/api/nodes` | POST, PATCH, GET `/:uuid`, DELETE `/:uuid` |
| Hosts | `/api/hosts` | POST, PATCH, GET `/:uuid`, DELETE `/:uuid` |
| System | `/api/system/health` | GET |

Auth: `POST /api/auth/login` with `{ username, password }` → `{ response: { accessToken } }`

### Three Resources

1. `remnawave_user` — VPN user (username, traffic limits, expiration, protocols)
2. `remnawave_node` — Xray server node
3. `remnawave_host` — Connection endpoint for subscriptions

All three use UUID as the primary identifier (unlike 3x-ui which uses int IDs).

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
