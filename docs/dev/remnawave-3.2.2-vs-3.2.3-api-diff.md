# Remnawave 3.2.2 vs 3.2.3 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.2.2...3.2.3`](https://github.com/remnawave/backend/compare/3.2.2...3.2.3)
and checking the matching
[`remnawave/frontend` flow](https://github.com/remnawave/frontend/compare/3.2.2...3.2.3).
The backend comparison contains 4 commits and 21 files with 195 additions and
22 deletions. Tag `3.2.3` resolves to backend commit
`e93afed2fb5cbfdfa9a7e0fe19fc35e4a005b978`; the previous `3.2.2` tag resolves
to `465bc090c271d707d5232d9faf2510cacba030b4`.

Remnawave 3.2.3 is additive for the existing Terraform CRUD surface, but it is
not only a compatibility-matrix bump. It introduces explicit snippet
synchronization, expands opaque Xray configuration behavior, and fixes node
plugin IPv6 validation.

## New snippet synchronization contract

The backend adds:

```text
POST /api/snippets/actions/sync
scope: snippets:sync
status: 202 Accepted
body: {"name":"<snippet-name>"}
```

The name uses the existing snippet constraints: 2–255 characters containing
letters, numbers, underscores, dashes, and spaces. The backend searches config
profile JSON for the name in:

- root-level `snippets[]`;
- `outbounds[].snippet`;
- `routing.rules[].snippet`;
- `routing.balancers[].snippet`.

It then force-starts every node using each affected config profile. The endpoint
therefore has a disruptive runtime side effect rather than merely updating
declarative database state.

API-token use of the opt-in behavior requires `system:metadata` for the patch
gate and `snippets:sync`, in addition to the normal `snippets:list`,
`snippets:create`, `snippets:update`, and `snippets:delete` lifecycle scopes.
The broader `system:read` scope also grants metadata access but is not the
least-privilege requirement.

The Remnawave 3.2.3 frontend calls this endpoint only after an explicit
confirmation following snippet update or deletion. The provider mirrors that
safety model with optional `remnawave_snippet.sync_nodes_on_change` rather than
restarting nodes silently. The option defaults to `false` and is gated on the
full backend version `3.2.3+` before any mutation.

On update, the provider updates the snippet and then synchronizes affected
nodes. On deletion, it deletes the snippet and then synchronizes by name. The
computed `sync_pending` phase (`none`, `update`, `delete`, or `recreate`) records
an incomplete opt-in operation before the mutation. Refresh remains read-only:
it reconciles whether the primary mutation committed, preserves the pending
phase when synchronization is still required, and does not remove state for a
pending deletion. `ModifyPlan` then creates a non-empty plan, and the normal
Update/Delete apply path retries only the missing synchronization step. Delete
recovery accepts the backend's 404 because the sync endpoint resolves profiles
by name after deletion.

Recovery is deliberately at-least-once: if the sync endpoint performs its work
but the response is lost, the next apply may queue a duplicate restart. This is
safer than reporting a clean plan while nodes can still use stale configuration,
and the provider exposes `sync_pending` so operators can observe the recovery
phase.

Existing configurations that omit `sync_nodes_on_change` keep their previous
behavior on every supported backend version.

## Other upstream changes

### Root-level snippets

Xray config processing now expands a root `snippets` array. Objects from named
snippets are merged into the config root without replacing protected keys or
existing root keys. Existing provider `config` and `snippet` JSON attributes
already preserve this opaque contract, so no additional Terraform schema is
required.

### Balancer-only snippet expansion

The Xray validator now expands `routing.balancers[].snippet` even when
`routing.rules` is absent. Version 3.2.2 nested balancer expansion under the
rules-presence condition, so a balancer-only config skipped expansion. This is
an upstream behavior fix already carried by the provider's opaque `config` and
`snippet` JSON attributes; no Terraform schema change is required.

### TLS cipher suites

Resolved TLS proxy configuration now carries `cipherSuites`; generated Xray
JSON writes `cipherSuites`, and Xray URI output writes the `cs` query parameter.
This is subscription rendering derived from config-profile JSON, not a new
Remnawave administrative CRUD field, so it is already carried through the
provider's existing opaque JSON fields.

### Node-plugin IPv6 validation

The node-plugin contract replaces the affected Zod IPv6 helpers with the
upstream regex source for IPv6 addresses and CIDRs. This fixes valid 6to4 values
in shared lists and plugin address fields. Provider acceptance coverage now
includes a `2002::/16` address and CIDR through `plugin_config`.

Package metadata moves to backend `3.2.3`, backend contract `3.2.3`, and node
plugins `0.6.3`.

## Published image

Docker Hub and GHCR resolve to the same multi-platform OCI index:

```text
remnawave/backend:3.2.3
sha256:bee71b9c3974e24007de4c13efd4aa6d5ec04b7fbf97cbe81095faac075a41b4
```

The provider matrix and default Docker Compose service pin both tag and digest.

## Verification

The following checks pass for this change set:

- unit tests with the race detector: 42.1% statement coverage;
- `go vet`, `golangci-lint` (0 issues), and `go build`;
- Terraform example validation and generated-documentation validation;
- `govulncheck`: no known reachable vulnerabilities;
- release artifact, release supply-chain, and repository policy test suites;
- 83 acceptance entry points against the exact 3.2.3 OCI index digest: `PASS`.

The 3.2.3 acceptance run specifically passes
`TestAccSnippetResourceSyncNodesOnChange` with an affected config profile and
node, `TestAccSnippetSyncRecoveryAcrossRefresh` with injected Update/Delete sync
failures, and `TestAccNodePluginIPv6_3_2_3`, in addition to the existing provider
regression suite.

## Compatibility matrix

The primary acceptance image moves from `remnawave/backend:3.2.2` to
`remnawave/backend:3.2.3`. Historical project policy keeps one latest-tested
patch per supported minor line, so CI retains 3.1.0, 3.0.0, 2.8.1, and 2.7.4.
Patch-aware unit and acceptance checks protect the 3.2.2 node-IP gate and the
new 3.2.3 snippet-sync gate.
