# Remnawave 3.4.4 vs 3.4.5 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.4.4...3.4.5`](https://github.com/remnawave/backend/compare/3.4.4...3.4.5)
(5 commits, 11 files, 63 additions, 23 deletions). The annotated tag `3.4.5`
(`70ba543c9c4044d58fc25bdad4db20defce9acf9`) resolves to backend commit
`010b365ab1fabea01192b5e6ade4e98e66ee1dbd`; the `3.4.4` tag resolves to
`b22970cc88481a7e278b5767721672a18f8b2ada`.

Remnawave 3.4.5 is a contract-compatible patch: no REST route, request DTO,
or response model changed — `libs/contract` is untouched. The Terraform
provider needs only a compatibility-matrix bump; exposing the new
`postStart` plugin section in `remnawave_node_plugin` is a follow-up (see
Additions). Update: implemented in
[#276](https://github.com/batonogov/terraform-provider-remnawave/pull/276).

## Additions

- **`SERVICE_SNI_VERIFICATION` env variable** (`ea12b0ba`): a new boolean
  (default `true`) in the panel configuration schema. When disabled, the
  panel skips SNI derivation on its mTLS connections to nodes
  (`axios.service.ts`, `mtls-agent.ts`). Server-side deployment
  configuration, not an API surface.
- **Post-start plugin schema** (`8274de8d`): the node-plugin configuration
  schema gains an optional `postStart` section with a `webhook` subsection
  that POSTs `{ "scope": "service", "event": "service.core_started", ... }`
  after Xray-core starts or restarts. Node-plugin configs are managed
  through the panel REST API, and the provider does manage them via
  `remnawave_node_plugin` — but its `plugin_config` whitelist
  (`canonicalNodePluginJSON`) accepts only `sharedLists`,
  `torrentBlocker`, `ingressFilter`, `egressFilter`, `connectionDrop`, and
  `preStart`, so a configuration containing `postStart` is currently
  rejected with `plugin_config contains unsupported key "postStart"`.
  Whitelisting `postStart` is a follow-up candidate; existing
  configurations without it are unaffected.

## Fixes

- **grpc transport property name** (`dec0fcca`): the xray-json subscription
  generator now emits `multiMode` instead of `mode` for grpc transport
  options. Affects rendered subscription output only; subscription content
  is opaque to the provider.
- **`id` filter handling in list queries** (`814fb054`): the users and
  torrent-blocker report repositories now treat numeric id filters as exact
  `BigInt` matches instead of `CAST(... AS TEXT) LIKE '%value%'` substring
  matches (`id` in users; `id`, `userId`, and `nodeId` in torrent-blocker
  reports). On an unparseable value the users repository now returns an
  empty result set (`WHERE false`) instead of silently ignoring the filter;
  the torrent-blocker fallback changes from `id IS NULL` to the equivalent
  `WHERE false` (the `telegramId` substring filter in users is unchanged
  apart from the same fallback swap). These filters back the admin UI data
  tables; the provider does not pass `id` filters to list endpoints, so
  behaviour it observes is unchanged.

## Chores

Version bump to 3.4.5 (`010b365a`) and the node-plugins package patch
version — no API impact.

## Published image

Docker Hub resolves to the following multi-platform OCI index:

```text
remnawave/backend:3.4.5
sha256:b16d724b90fd7c9fec2df04bd28938a671cafc62894105068e11550ee3449c56
```

Runnable platforms inspected before the compatibility bump:

- `linux/amd64` — `sha256:09c373529b2b0be65bc23443da1f87289450ee56a65f8b2276617ce33e0191d9`
- `linux/arm64` — `sha256:705c68eda8f1969206d8bf16163f0fa95cd48969e3dd7b3f5125426d862eb208`

The additional `unknown/unknown` manifests are OCI attestations, not runtime
platforms.

## Verification

The complete `TestAcc*` suite passed against the exact 3.4.5 index digest:

```text
PASS
ok  github.com/batonogov/terraform-provider-remnawave/provider  55.211s
```

The import-only WebAuthn passkey test retained its expected skip because no
pre-existing passkey fixture was supplied, and the pre-3.4 host-squads
rejection test skipped because the default panel is now 3.4.5.

## Compatibility matrix

The default acceptance image moves from `remnawave/backend:3.4.4` to
`remnawave/backend:3.4.5`. The supported minor line remains `3.4.x`; CI
retains 3.3.2, 3.3.1, 3.2.3, 3.1.0, 3.0.0, 2.8.1, and 2.7.4 to protect older
supported API contracts.

The ruleset file in this pull request does not apply itself: the live
GitHub ruleset still requires `Acceptance Tests (3.4.4)`, a context the new
matrix never produces. Run `scripts/configure-repository-security.sh` from
this branch before merging so the live ruleset picks the
`Acceptance Tests (3.4.5)` context up.
