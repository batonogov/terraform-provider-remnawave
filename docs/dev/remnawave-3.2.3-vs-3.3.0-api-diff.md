# Remnawave 3.2.3 vs 3.3.0 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.2.3...3.3.0`](https://github.com/remnawave/backend/compare/3.2.3...3.3.0).
The comparison contains 20 commits and 159 files with 4,351 additions and 749
deletions. Tag `3.3.0` resolves to backend commit
`542f131487f24bcc0121ae0d1cdd58cea8fd9d86`; `3.2.3` resolves to
`e93afed2fb5cbfdfa9a7e0fe19fc35e4a005b978`.

Remnawave 3.3.0 adds declarative node integrations and global node-plugin
shared lists, and extends the existing node and host contracts. It also changes
the node-plugin editor contract: `sharedLists` remains in effective responses
but is no longer accepted inside `pluginConfig` updates.

## Provider surface

### Node integrations

The backend adds CRUD and list routes under `/api/node-integrations`. An
integration contains `uuid`, `name`, nullable `description`, and an opaque
`config` object. Updates can request `restartNodes`; deletion removes the
integration from assigned nodes and restarts affected profiles.

The provider adds:

- `remnawave_node_integration` with optional `restart_nodes_on_update`;
- `remnawave_node_integrations` for inventory and brownfield discovery;
- `remnawave_node.integration_uuids` and the matching computed field in the
  nodes data source.

All configured operations fail before mutation on a backend older than 3.3.
The node assignment has the backend limit of 20 integration UUIDs.

### Global shared lists

Shared lists move from embedded node-plugin configuration to global records
under `/api/node-plugins/shared-lists`. Names omit the runtime `ext:` prefix and
contain 2–255 letters, digits, underscores, or dashes. The list config is either
an `ipList` containing IP/CIDR strings or an `asList` containing numeric ASNs.

The provider adds:

- `remnawave_shared_list`, keyed and imported by name;
- `remnawave_shared_lists`, which exposes the backend's name/type/item-count
  preview response.

For compatibility, node-plugin PATCH payloads on 3.3 omit `sharedLists`, while
refresh preserves the value already represented by the node-plugin resource.
This prevents the backend's injected effective-list response from creating
perpetual drift. New global contents should be owned by
`remnawave_shared_list`; the inline field remains the 2.x–3.2 wire contract and
a compatibility view for upgraded state.

The explicit plugin and shared-list synchronization routes are intentionally
not exposed yet. They are imperative, disruptive operations and need the same
retry/recovery design used by snippet synchronization before becoming provider
features.

### Host mapper

Host create, update, and response models add `mapper`. It is an object with
optional `xrayJson`, `mihomo`, `base64`, and `singbox` operation arrays. Each
operation copies, sets, or unsets a generated client-config value.

`remnawave_host.mapper` carries this contract as normalized opaque JSON and is
gated on Remnawave 3.3+. Keeping it opaque matches the existing config-profile,
snippet, transport, and response-rule JSON strategy and allows additive mapper
operations without a provider schema release.

### Other upstream changes

- Subscription response-rule modifications add `respondWithRemarks`. The
  provider's existing opaque `response_rules` JSON already carries it.
- Connections add asynchronous node geocheck request/result routes. They are a
  live diagnostic job, not declarative state, and are not exposed in this
  compatibility change.
- Subscription generation and resolved-proxy schemas are refactored, including
  sing-box generation. Existing opaque Xray/subscription JSON fields need no
  schema changes.
- Backend contract package metadata is `3.4.2`; the node-plugin contract is
  `0.7.0`; the application version is `3.3.0`.

## Published image

Docker Hub and GHCR resolve the multi-platform OCI index to:

```text
remnawave/backend:3.3.0
sha256:a93047b52f7ef6368b2b776e0cd728ae7548bd50fe8ca6f782277e5cbb6dbab1
```

The default Docker Compose service and compatibility matrix pin both the tag
and digest. The historical 3.2.3 entry remains because 3.3.0 starts a new
supported minor line.

## Verification target

The change is covered by:

- client method/path/body contract tests for all new CRUD/list operations;
- version-gate and model conversion unit tests;
- node-plugin 3.3 editor/effective-response compatibility tests;
- node, host, node-integration, and shared-list acceptance lifecycles on 3.3.0;
- the full historical acceptance matrix on 3.2.3, 3.1.0, 3.0.0, 2.8.1, and
  2.7.4.
