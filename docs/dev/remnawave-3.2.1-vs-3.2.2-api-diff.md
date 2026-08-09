# Remnawave 3.2.1 vs 3.2.2 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.2.1...3.2.2`](https://github.com/remnawave/backend/compare/3.2.1...3.2.2)
(8 commits, 30 files, 87 additions, 18 deletions). Remnawave 3.2.2 extends the
existing node create, update, and response contracts with managed IP metadata.
The Terraform provider therefore requires a schema and model update rather than
only a compatibility-matrix bump.

## Provider-relevant changes

### Managed node IPs

Node requests and responses now include an `ips` array with up to 64 objects:

```json
{
  "ip": "192.0.2.10",
  "status": "MANAGEMENT"
}
```

`ip` must be an IPv4 or IPv6 address. `status` is one of:

- `INBOUND`
- `OUTBOUND`
- `MANAGEMENT`
- `TRANSIT`
- `MONITORING`
- `RESERVE`
- `BLOCKED`
- `FLAGGED`
- `DEPRECATED`
- `UNKNOWN`

The backend stores the array in a new non-null `nodes.ips` JSONB column with an
empty-array default and exposes it through node create, update, single-node, and
list responses.

The provider exposes `ips` as an optional/computed nested set on
`remnawave_node` and as a computed nested set on `remnawave_nodes`. Provider-side
validation enforces the address format, status enum, and 64-item limit. Because
the contract was introduced in a patch release, configuring a non-empty set on a
backend older than 3.2.2 fails before any create or update request is sent.
Existing configurations that omit `ips` remain compatible with every supported
backend line.

### Torrent Blocker webhook

The Torrent Blocker node-plugin schema adds the optional URL field
`webhookUrl`. `remnawave_node_plugin.plugin_config` already accepts and
canonically preserves arbitrary plugin-specific JSON keys, so no provider schema
change is required for this addition.

### VLESS UUID validation

User create and response schemas switch VLESS identifiers from Zod `uuid()` to
`guid()` validation. Both serialize as the existing `vlessUuid` string field;
the provider request and state models do not change.

## Backend-only changes

- `REDIS_USERNAME` is now accepted by the application, CLI, raw cache, queue,
  and seed cleanup Redis clients.
- Updating an external squad now invalidates cached template-name entries before
  template synchronization.
- HWID device ranking adds user ID as a deterministic secondary sort key.
- The runtime and build images move from Node.js 24.18 to 24.19.
- Package metadata is bumped to backend 3.2.2, contract 3.2.2, and node-plugins
  0.6.2.

These changes do not alter another resource or data-source contract exposed by
the provider.

## Published image

Docker Hub and GHCR resolve to the same multi-platform OCI index:

```text
remnawave/backend:3.2.2
sha256:44607a941eb1343a3975e5cc77b65207c597c3af4d00b80e4e32ebd48e73abd5
```

Runnable platforms inspected before the compatibility bump:

- `linux/amd64` — `sha256:5df9bafb2b486e376ccea56b2b6e8a38c749de829cb4e613571c447b19fc7d14`
- `linux/arm64` — `sha256:8c47ef46b3bee8fe954a33390aba3bf243aa0edc1759b25d4130314f8e0a8f7b`

The additional `unknown/unknown` manifests are OCI attestations, not runtime
platforms.

## Verification

The complete `TestAcc*` suite passed against the exact 3.2.2 index digest:

```text
PASS
ok  github.com/batonogov/terraform-provider-remnawave/provider  121.450s
```

The import-only WebAuthn passkey test retained its expected skip because no
pre-existing passkey fixture was supplied. Node resource create, update, import,
and node-list data-source acceptance paths exercised the new `ips` contract.

## Compatibility matrix

The default acceptance image moves from `remnawave/backend:3.2.1` to
`remnawave/backend:3.2.2`. The supported minor line remains `3.2.x`; CI retains
3.1.0, 3.0.0, 2.8.1, and 2.7.4 to protect older supported API contracts.
