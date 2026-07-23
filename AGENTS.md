# AGENTS.md

This file provides guidance to AI coding agents when working with this repository.

## Project

Terraform provider for [Remnawave](https://docs.rw) — a proxy management panel
built on Xray-core. Go with `terraform-plugin-framework`.
Module: `github.com/batonogov/terraform-provider-remnawave`.
Registry: `batonogov/remnawave`. All provider code lives in `provider/`.

The Remnawave backend (`github.com/remnawave/backend`) is a NestJS TypeScript
application with a clean REST API. The panel uses PostgreSQL + Redis (Valkey).

**Compatibility:** Remnawave v2.7.x and v2.8.x. Docker Compose and
acceptance tests default to the `remnawave/backend:2.8.1` image pinned by
digest; CI runs a second matrix entry against `remnawave/backend:2.7.4`.
All compose images are pinned by `sha256` digest for reproducibility. To
run an explicit compatibility check against a different build, override
both the tag and its digest, e.g. `REMNAWAVE_VERSION=2.9.0
REMNAWAVE_DIGEST=sha256:<digest>`.

The client auto-detects the server version via `/api/system/metadata` on
the first API-token operation. On 2.7.x backends the `remnawave_api_token`
resource transparently uses the legacy `tokenName` request field and
`apiKeys[]` response shape instead of the 2.8.x `name`/`expiresInDays`/
`scopes` request and `tokens[]` response. No user configuration is
required. All other resources/data sources are forward-compatible: 2.7.x
Zod validation strips unknown 2.8.x fields without error.

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

### Resources (26)

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
| `remnawave_host_bulk_action` | `resource_host_bulk_action.go` | `/api/hosts/bulk/{enable,disable,delete}` |
| `remnawave_user_bulk_action` | `resource_user_bulk_action.go` | `/api/users/bulk/*` |
| `remnawave_node_bulk_action` | `resource_node_bulk_action.go` | `/api/nodes/bulk-actions` |
| `remnawave_node_action` | `resource_node_action.go` | `/api/nodes/:uuid/actions/{enable,disable,restart,reset-traffic}` |
| `remnawave_drop_connections` | `resource_drop_connections.go` | `/api/ip-control/drop-connections` |
| `remnawave_user_action` | `resource_user_action.go` | `/api/users/:uuid/actions/{enable,disable,reset-traffic,revoke}` |
| `remnawave_passkey` | `resource_passkey.go` | `/api/passkeys` |

### Data Sources (23)

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
| `remnawave_host_tags` | `data_source_host_tags.go` | `/api/hosts/tags` |
| `remnawave_user_ips` | `data_source_user_ips.go` | `/api/ip-control/fetch-ips/:uuid` |
| `remnawave_passkeys` | `data_source_passkeys.go` | `/api/passkeys` |

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

### Post-merge sync

When the user says that a PR was merged (for example, `смерджил`), immediately
switch to `main` and fast-forward it from the remote:

```sh
git switch main
git pull --ff-only origin main
git clean -f
```

Do not delete the feature branch unless the user explicitly asks. `git clean -f`
removes untracked duplicate/generated files such as `docs/* 2.md`; preview with
`git clean -nd` first when other untracked files may be present.

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

### Continuous Integration

`.github/workflows/ci.yml` runs on every pull request and push to `main`:

| Job | Checks |
| --- | --- |
| Lint | `golangci-lint run` |
| Build | `go build ./...` |
| Unit Tests | `go test ./provider -skip TestAcc`, race detector, **30% coverage floor** |
| Documentation | `terraform fmt -check` on examples; `tfplugindocs generate/validate`; fails if `docs/` drifts |
| Acceptance Tests | Full `docker compose` panel lifecycle + `TestAcc*` — **matrix** against both 2.8.1 (default) and 2.7.4 |

All GitHub Actions across the repo **must be pinned by commit SHA**
(see `release-please.yml`); Dependabot keeps them current. Do not switch
workflow steps to floating tags like `@v7`.

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

1. A push to `main` runs the normal CI workflow. Only a successful, completed
   CI run for the current `main` SHA can trigger the release workflow.
2. The release workflow verifies every CI job, rejects stale or mismatched
   revisions, and checks the embedded Go `vcs.revision` before granting write
   permissions to `release-please`.
3. `release-please` groups Conventional Commits since the last release into a
   release PR titled `chore(main): release X.Y.Z`.
4. Merging that PR repeats the exact-SHA gate before creating the `vX.Y.Z` tag
   and GitHub Release. `.release-please-manifest.json` remains the source of
   truth for the next version.
5. `release_created == true` triggers the `goreleaser` job. It verifies that
   the checkout, tag target, Release Please output, and CI-tested SHA are
   identical before importing the GPG key, then builds and attaches archives,
   `SHA256SUMS`, and `SHA256SUMS.sig`. The Terraform Registry picks up the
   release automatically.

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
- Copyable `required_providers` examples use a pessimistic patch-line
  constraint (`~> X.Y.0`). When a release changes the minor or major line,
  update the constraint in `README.md`, `examples/provider.tf`, and
  `examples/provider/provider.tf`, then run `task docs` to regenerate
  `docs/index.md`. Patch releases within the same minor line need no update.

### Required release environment secrets

The goreleaser job fails without these secrets:

- `RELEASE_GPG_PRIVATE_KEY` — keypair used to detach-sign `SHA256SUMS` (binary
  signature, not ASCII-armored).
- `RELEASE_GPG_PASSPHRASE` — passphrase for the key (cached before signing;
  goreleaser itself cannot prompt interactively).

Store both only on the protected `release` GitHub Environment, not as
repository-level secrets. The Environment requires an independent deployment
approval before the GoReleaser job can access them.

### Build contract

- `terraform-registry-manifest.json` declares `protocol_versions: ["6.0"]`
  (Plugin Framework default; matches `providerserver.Serve` in `main.go`).
- `main.version` (`main.go`) is injected at build time via goreleaser ldflags
  (`-X main.version`); locally built binaries report `dev`.
- Builds are reproducible: `-trimpath` + `mod_timestamp`. Release targets are:
  Linux (`amd64`, `arm64`, `arm`, `386`), macOS (`amd64`, `arm64`), and Windows
  and FreeBSD (`amd64`, `386`).
- `compat-versions.json` records the supported Remnawave backend versions. Keep
  it in sync with the **Compatibility** note in `## Project` when bumping the
  target line. CI acceptance tests use it as the source of truth for the
  version matrix.

### Pre-release gate

The release workflow is triggered by `workflow_run` and enforces successful
lint, build, unit, documentation, release-gate, and compatibility-matrix
acceptance jobs for the exact current `main` SHA. Failed, cancelled, skipped,
missing, duplicate, or stale results block both Release Please and GoReleaser.
Ordinary CI jobs have read-only permissions and cannot access release or GPG
credentials.
