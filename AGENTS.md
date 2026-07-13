# AGENTS.md

This file provides guidance to AI coding agents when working with this repository.

## Project

Terraform provider for [Remnawave](https://docs.rw) — a proxy management panel
built on Xray-core. Go with `terraform-plugin-framework`.
Module: `github.com/batonogov/terraform-provider-remnawave`.
Registry: `batonogov/remnawave`. All provider code lives in `provider/`.

The Remnawave backend (`github.com/remnawave/backend`) is a NestJS TypeScript
application with a clean REST API. The panel uses PostgreSQL + Redis (Valkey).

**Compatibility:** Remnawave v2.8.x. Docker Compose and acceptance tests default
to the `remnawave/backend:2.8.0` image pinned by digest. All compose images
are pinned by `sha256` digest for reproducibility. To run an explicit
compatibility check against a different build, override both the tag and its
digest, e.g. `REMNAWAVE_VERSION=2.9.0 REMNAWAVE_DIGEST=sha256:<digest>`.

## Commands

| Command | Description |
| --- | --- |
| `task build` | Build provider binary |
| `task fmt` | `gofmt -w provider/*.go` |
| `task vet` | `go vet ./...` |
| `task lint` | `golangci-lint run` |
| `task test:unit` | Unit tests (no Docker/Terraform needed) |
| `task test:coverage` | Unit tests with race detection and coverage |
| `task test:acc` | Acceptance tests (Docker lifecycle included) |
| `task docs` | Format examples and regenerate Registry docs |
| `task docs:check` | Validate examples/docs and detect generated drift |

## Architecture

### HTTP Client (`client.go`)

JWT Bearer auth. Two modes:
- **API token** (static JWT) — provided directly, no login needed
- **Username/password** — provider calls `POST /api/auth/login` to obtain JWT

Auto re-authenticates on 401 (unless using static API token). Requests using a
login-issued JWT set `X-Remnawave-Client-Type: browser`, as required by
Remnawave v2.8 proxy checks; static API-token requests do not set it.
All responses are wrapped in `{ "response": <data> }` envelope — `decodeResponse`
unwraps automatically.

`resolvePath` splits path and query string to avoid URL-encoding `?`.

Panel branding PATCH payloads must include both `title` and `logoUrl` keys when
`brandingSettings` is present. Remnawave accepts `null` values, so do not add
`omitempty` to those nested JSON fields.

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

Run the complete Docker lifecycle and suite with `task test:acc`.

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
- HTTP client unit tests use `httptest` and cover auth, 401 retry, response
  decoding, errors, proxy headers, and every exported API operation.
- CI runs unit tests with the race detector and enforces a 30% unit coverage
  floor. Resource/data-source CRUD paths are additionally covered by the real
  panel acceptance suite.
- Acceptance tests: `TestAccXxx`, `terraform-plugin-testing`,
  `ProtoV6ProviderFactories`.

### Documentation

- Registry docs in `docs/` are generated by `tfplugindocs` from schemas and
  examples. Do not hand-edit generated schema sections.
- Examples follow the standard paths under `examples/provider/`,
  `examples/resources/<type>/`, and `examples/data-sources/<type>/`.
- Run `task docs` after schema/example changes and `task docs:check` before a
  PR. The generator is pinned in the command and intentionally not added to the
  provider's runtime `go.mod` dependency graph.

### Dependencies

- Prefer the Go standard library and existing modules.
- Add production dependencies only when clearly necessary and document the
  reason in the PR. Keep build-only tools out of `go.mod` when practical.
- Dependabot checks Go modules and GitHub Actions weekly; minor/patch updates
  are grouped, while major updates remain separate for review.

### Auth

- API token auth preferred over username/password (avoids login on every plan).
- Panel env: `IS_DOCS_ENABLED=true` enables Swagger at `/docs` for API exploration.
- Panel env: `NODE_ENV=development` disables ProxyCheckMiddleware for direct access.

## Releases

Releases are fully automated via `.github/workflows/release-please.yml` — **never
tag or publish manually.**

### Flow

1. A push to `main` runs `release-please`, which groups Conventional Commits
   since the last release into a release PR titled `chore(main): release X.Y.Z`.
2. Merging that PR creates the `vX.Y.Z` tag, a GitHub Release, and updates
   `.release-please-manifest.json` (source of truth for the next version) and
   `CHANGELOG.md`.
3. `release_created == true` triggers the `goreleaser` job, which builds the
   provider for every supported platform, attaches archives + `SHA256SUMS` + a
   GPG-detached `SHA256SUMS.sig` to the release, and the Terraform Registry
   picks the release up automatically.

### Versioning

- Tags are strict Semantic Versioning with a `v` prefix (`v1.2.3`). The
  Registry resolves, sorts, and constraints versions by SemVer. Prereleases
  use a hyphen (`v1.2.3-pre`) and are never selected automatically.
- Bump level is driven by commit type: `feat:` → minor, `fix:` → patch,
  `feat!:` / `BREAKING CHANGE:` → major. `docs:`, `test:`, `ci:`, and `chore:`
  commits are excluded from the changelog and do not on their own cut a release.
- **Never modify, re-tag, or replace a released version** — it breaks the
  published checksums for existing users. Ship a new version instead.
- A tag must not share its name with a branch.

### Required repository secrets

The goreleaser job fails without these secrets:

- `GPG_PRIVATE_KEY` — keypair used to detach-sign `SHA256SUMS` (binary
  signature, not ASCII-armored).
- `GPG_PASSPHRASE` — passphrase for the key (cached before signing;
  goreleaser itself cannot prompt interactively).

### Build contract

- `terraform-registry-manifest.json` declares `protocol_versions: ["6.0"]`
  (Plugin Framework default; matches `providerserver.Serve` in `main.go`).
- `main.version` (`main.go`) is injected at build time via goreleaser ldflags
  (`-X main.version`); locally built binaries report `dev`.
- Builds are reproducible: `-trimpath` + `mod_timestamp`. Multi-platform matrix:
  linux/darwin/windows/freebsd × amd64/arm64/arm/386.
- `compat-versions.json` records the supported Remnawave backend versions. Keep
  it in sync with the **Compatibility** note in `## Project` when bumping the
  target line; it is advisory (not yet enforced by CI).

### Pre-release gate

Before merging a release PR, confirm CI on `main` is green — the acceptance
suite (`task test:acc`) must pass against the pinned backend image.
