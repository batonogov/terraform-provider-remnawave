# Remnawave 3.0.0 vs 3.1.0 — API Contract Differences

## Summary

These findings come from diffing `github.com/remnawave/backend` tags `3.0.0`
and `3.1.0`. The release is additive for the Terraform provider: existing 3.0
routes and request payloads remain compatible.

## 1. Nodes — numeric ID added

`NodesSchema` and `NodeResponseModel` add a numeric `id` alongside the existing
`uuid`. Node routes and mutations continue to use the UUID.

Provider impact:

- `remnawave_node.id` is a computed integer on 3.1+ and `null` on older panels.
- `remnawave_nodes.nodes[*].id` exposes the same value in the list data source.
- The existing `uuid` remains the Terraform import and lifecycle identifier.

## 2. Subscription request history — SRR fields added

Both the global and per-user history record schemas add:

| Field | Type | Meaning |
| --- | --- | --- |
| `srrRuleName` | `string \| null` | Matched subscription response rule name |
| `srrResponseType` | `string` | Response type selected by the rule engine |

The `remnawave_subscription_request_history` data source returns the response
as raw JSON, so both fields are available automatically on 3.1+ without a
version-specific schema or configuration change.

## 3. Node plugins — pre-start stage added

`NodePluginSchema` adds an optional `preStart` section. Its first supported
operation is stale Unix socket cleanup before Xray starts:

```json
{
  "preStart": {
    "enabled": true,
    "cleanupSockets": {
      "enabled": true,
      "files": ["/dev/shm/*.sock"]
    }
  }
}
```

The provider accepts `preStart` in `plugin_config` on Remnawave 3.1+ and returns
a clear version error instead of allowing older panels to strip the field and
produce inconsistent Terraform state.

## 4. Other changes

- Failed logins now include additional backend logging; the HTTP contract is
  unchanged.
- Bulk user deletion emits previously missing events; routes and payloads are
  unchanged.
- Node plugin validation errors now include their detailed Zod message while
  retaining the same error code.

## Compatibility matrix

The default acceptance image moves to `remnawave/backend:3.1.0`. CI retains
3.0.0, 2.8.1, and 2.7.4 entries to ensure the additive fields do not regress
older supported minor lines.
