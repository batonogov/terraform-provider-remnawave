# CLAUDE.md

This file provides guidance to AI coding agents when working with this repository.

## Project

Terraform provider for [Remnawave](https://docs.rw) — a proxy management panel
built on Xray-core. Go with `terraform-plugin-framework`.
Module: `github.com/batonogov/terraform-provider-remnawave`.
Registry: `batonogov/remnawave`. All provider code lives in `provider/`.

The Remnawave backend (`github.com/remnawave/backend`) is a NestJS TypeScript
application with a clean REST API. The panel uses PostgreSQL + Redis (Valkey).

**Compatibility:** Remnawave v2.7.x, v2.8.x, v3.0.x, v3.1.x, v3.2.x, and v3.3.x.
Docker Compose and acceptance tests default to the `remnawave/backend:3.3.2`
image pinned by digest; CI runs matrix entries against `remnawave/backend:3.3.1`,
`remnawave/backend:3.2.3`, `remnawave/backend:3.1.0`, `remnawave/backend:3.0.0`,
`remnawave/backend:2.8.1`, and `remnawave/backend:2.7.4`. Remnawave 3.3.1 stays in
the matrix even though 3.3.2 supersedes it, because 3.3.1 is the only version
that injects a `torrentBlocker.rulePlacement` default. All compose images
are pinned by `sha256` digest for reproducibility. To run an explicit
compatibility check against a different build, override both the tag and its
digest, e.g. `REMNAWAVE_VERSION=3.3.2 REMNAWAVE_DIGEST=sha256:<digest>`.

The client auto-detects the server version via `/api/system/metadata` on
the first version-dependent operation. Version-specific behaviour:

- **2.7.x**: `remnawave_api_token` uses legacy `tokenName` request field
  and `apiKeys[]` response shape. Hosts limited to a single `tag` field.
- **2.8.x**: `name`/`expiresInDays`/`scopes` token request. Hosts use
  `tags[]` array. All 2.7.x fields remain available.
- **3.0.x**: User routes use numeric `id` instead of `uuid`. The
  `ip-control` module is renamed to `connections` (paths changed).
  Subscription settings dropped 6 fields (`profileTitle`, `supportLink`,
  `profileUpdateInterval`, `isProfileWebpageUrlEnabled`, `happAnnounce`,
  `happRouting`). External squad `responseHeaders` split into
  `responseHeadersAdd` + `responseHeadersRemove`. The provider exposes the new
  user action `extend_expiration`; other backend-only endpoints introduced in
  3.0 are not part of the provider surface yet.
- **3.1.x**: Nodes expose a numeric `id` in addition to `uuid`. Subscription
  request history records add `srrRuleName` and `srrResponseType`; the raw JSON
  data source exposes both fields without version-specific configuration.
- **3.2.x**: Additive only. Adds a read-only `GET /api/system/configuration`
  endpoint that returns server-side environment configuration (webhook,
  notification thresholds, service toggles, short UUID length, subscription
  public domain, usage floor). Config-profile outbound validation is extracted
  into a dedicated step but remains equivalent; no provider changes required.
  The new endpoint is not yet part of the provider surface. Version 3.2.1 is a
  contract-compatible patch that removes invalid legacy API-token UUIDs during
  migration and changes only subscription generation internals. Version 3.2.2
  adds node `ips` request/response data, an optional Redis username, node-plugin
  webhook configuration, strict VLESS UUID validation, and internal cache/query
  fixes. The provider exposes node `ips` only on 3.2.2+; plugin `webhookUrl`
  remains available through the existing `plugin_config` JSON contract.
- **3.3.x**: Adds host mapper operations, reusable node integrations and node
  `integrationUuids`, global node-plugin shared lists, explicit plugin/list sync,
  connection geocheck jobs, and `respondWithRemarks` in subscription response
  rules. The provider exposes host mappers, node integration CRUD/list and node
  assignments, and global shared-list CRUD/list. Response rules remain opaque
  JSON. Geocheck and explicit plugin/list sync are not yet provider surfaces.
  Node-plugin writes omit the legacy inline `sharedLists`; reads preserve the
  configured compatibility view while global list contents are managed through
  `remnawave_shared_list`. Versions 3.3.1 and 3.3.2 are contract-compatible
  patches, but not behavior-neutral: 3.3.1 adds the optional
  `torrentBlocker.rulePlacement` key (0-1000) to the opaque `plugin_config`
  JSON *with* a schema default of `0`, and 3.3.2 removes that default. The
  node-plugin schema is not strict and the backend stores the parsed config, so
  3.3.0 silently strips the key while 3.3.1 injects it. The provider gates the
  nested key on 3.3.1+ and drops a returned `rulePlacement` that the
  configuration did not set (`alignNodePluginRulePlacement`); it must never
  materialize a default of its own, which would drift against 3.3.2. The
  patches also fix node-plugin reordering, which previously wrote `viewPosition`
  to the `externalSquads` table, and change an internal Telegram notification
  URL.

Existing configurations require no changes — the provider transparently
adapts. The optional node `ips` attribute requires Remnawave 3.2.2 or later;
host `mapper`, node `integration_uuids`, node integrations, and global shared
lists require Remnawave 3.3 or later; `torrentBlocker.rulePlacement` in
`plugin_config` requires Remnawave 3.3.1 or later.

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

### Resources (28)

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
| `remnawave_node_integration` | `resource_node_integration.go` | `/api/node-integrations` |
| `remnawave_shared_list` | `resource_shared_list.go` | `/api/node-plugins/shared-lists` |
| `remnawave_api_token` | `resource_api_token.go` | `/api/tokens` |
| `remnawave_infra_provider` | `resource_infra_provider.go` | `/api/infra-billing/providers` |
| `remnawave_billing_node` | `resource_billing_node.go` | `/api/infra-billing/nodes` |
| `remnawave_billing_history` | `resource_billing_history.go` | `/api/infra-billing/history` |
| `remnawave_subpage_config` | `resource_subpage_config.go` | `/api/subscription-page-configs` |
| `remnawave_user_metadata` | `resource_user_metadata.go` | `/api/metadata/user/:identifier` |
| `remnawave_node_metadata` | `resource_node_metadata.go` | `/api/metadata/node/:uuid` |
| `remnawave_hwid_device` | `resource_hwid_device.go` | `/api/hwid/devices` |
| `remnawave_host_bulk_action` | `resource_host_bulk_action.go` | `/api/hosts/bulk/{enable,disable,delete}` |
| `remnawave_user_bulk_action` | `resource_user_bulk_action.go` | `/api/users/bulk/*` |
| `remnawave_node_bulk_action` | `resource_node_bulk_action.go` | `/api/nodes/bulk-actions` |
| `remnawave_node_action` | `resource_node_action.go` | `/api/nodes/:uuid/actions/{enable,disable,restart,reset-traffic}` |
| `remnawave_drop_connections` | `resource_drop_connections.go` | 2.x: `/api/ip-control/drop-connections`<br>3.0+: `/api/connections/drop` |
| `remnawave_user_action` | `resource_user_action.go` | `/api/users/:identifier/actions/{enable,disable,reset-traffic,revoke,extend}` |
| `remnawave_passkey` | `resource_passkey.go` | `/api/passkeys` |

### Data Sources (27)

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
| `remnawave_bandwidth_stats_user` | `data_source_bandwidth.go` | `/api/bandwidth-stats/users/:identifier` |
| `remnawave_bandwidth_realtime` | `data_source_misc_stats.go` | `/api/bandwidth-stats/nodes/realtime` |
| `remnawave_system_bandwidth_stats` | `data_source_misc_stats.go` | `/api/system/stats/bandwidth` |
| `remnawave_system_nodes_stats` | `data_source_misc_stats.go` | `/api/system/stats/nodes` |
| `remnawave_subscriptions` | `data_source_subscriptions.go` | 2.x: `/api/subscriptions/by-uuid/:uuid`<br>3.0+: `/api/subscriptions/by-id/:id` |
| `remnawave_subscription_request_history` | `data_source_subscription_request_history.go` | `/api/subscription-request-history` |
| `remnawave_subscription_request_history_stats` | `data_source_misc_stats.go` | `/api/subscription-request-history/stats` |
| `remnawave_connection_keys` | `data_source_misc_stats.go` | `/api/subscriptions/connection-keys/:identifier` |
| `remnawave_hwid_stats` | `data_source_hwid.go` | `/api/hwid/devices/stats` |
| `remnawave_hwid_top_users` | `data_source_hwid.go` | `/api/hwid/devices/top-users` |
| `remnawave_host_tags` | `data_source_host_tags.go` | `/api/hosts/tags` |
| `remnawave_user_ips` | `data_source_user_ips.go` | 2.x: `/api/ip-control/fetch-ips/:uuid`<br>3.0+: `/api/connections/by-user/:id` |
| `remnawave_passkeys` | `data_source_passkeys.go` | `/api/passkeys` |
| `remnawave_internal_squads` | `data_source_internal_squads.go` | `/api/internal-squads` |
| `remnawave_external_squads` | `data_source_external_squads.go` | `/api/external-squads` |
| `remnawave_node_integrations` | `data_source_node_integrations.go` | `/api/node-integrations` |
| `remnawave_shared_lists` | `data_source_shared_lists.go` | `/api/node-plugins/shared-lists` |

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
| Acceptance Tests | Full `docker compose` panel lifecycle + `TestAcc*` — **matrix** against 3.3.2 (default), 3.3.1, 3.2.3, 3.1.0, 3.0.0, 2.8.1, and 2.7.4 |

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
   and a **draft** GitHub Release. `.release-please-manifest.json` remains the
   source of truth for the next version; `release-please-config.json` forces
   immediate tag creation so the draft has a stable source revision.
5. `release_created == true` triggers the protected `goreleaser` job. It
   verifies that the checkout, tag target, Release Please output, and CI-tested
   SHA are identical before importing the GPG key. It then builds archives,
   per-archive SPDX SBOMs, signed checksums, and GitHub/Sigstore provenance.
6. The workflow publishes the draft only after every archive, SBOM, checksum,
   embedded VCS revision, provenance subject, source revision, and workflow
   identity has been verified. The Terraform Registry then picks up the
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
- Copyable `required_providers` examples pin the **major** line
  (`version = "~> X.0"`), so users receive non-breaking minor and patch
  releases without editing their configuration. The constraint is owned
  end-to-end by `release-please` via the `# x-release-please-major`
  annotation: its generic updater rewrites only the first integer on the line
  and leaves the trailing `.0` intact, so the string changes on a major bump
  and never on a minor or patch bump. Never use `# x-release-please-version`
  here — it rewrites the patch component and breaks the invariant that
  `scripts/check-docs-inventory.sh` enforces. Both the annotation and the
  checker derive from `.release-please-manifest.json`, so they cannot drift
  apart. `scripts/test-docs-inventory-version-bumps.sh` proves this by
  replaying patch, minor, and major bumps through the updater's own
  replacement semantics. The constraint appears in `README.md` (×2),
  `examples/getting-started/main.tf`, `examples/provider.tf`, and
  `examples/provider/provider.tf`; `docs/index.md` inherits it from the last
  file through `task docs`.

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
  Linux (`amd64`, `arm64`, `arm`, `386`), macOS (`amd64`, `arm64`), Windows
  (`amd64`, `arm64`, `386`), and FreeBSD (`amd64`, `386`).
  `release-targets.json` is the machine-readable archive contract; keep it in
  sync with `.goreleaser.yml`.
- Release builds must start from a clean detached tag checkout. `/dist/` is the
  only ignored in-worktree output. The release workflow builds once without
  publishing, verifies every archive's checksum and embedded Go VCS/module
  metadata, then requires the published build to reproduce the same archive
  checksums.
- Every archive has a `<archive>.spdx.json` SBOM generated by the pinned Syft
  version. The Terraform Registry checksum contains only provider archives and
  the Registry manifest. Final archives and SBOMs are subjects of one
  GitHub/Sigstore SLSA provenance bundle, verified against the tag commit and
  release workflow before the draft is published. Keep `id-token: write` and
  `attestations: write` scoped to the protected release job.
- `compat-versions.json` records the supported Remnawave backend versions. Keep
  it in sync with the **Compatibility** note in `## Project` when bumping the
  target line. CI acceptance tests use it as the source of truth for the
  version matrix.

### Pre-release gate

The release workflow is triggered by `workflow_run` and enforces successful
lint, build, unit, documentation, release-gate, release-artifact,
release-supply-chain, and compatibility-matrix acceptance jobs for the exact
current `main` SHA. Failed, cancelled, skipped, missing, duplicate, or stale
results block both Release Please and GoReleaser. Ordinary CI jobs have
read-only permissions and cannot access release, attestation, or GPG
credentials.
