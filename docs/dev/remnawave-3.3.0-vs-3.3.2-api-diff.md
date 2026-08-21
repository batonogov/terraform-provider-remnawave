# Remnawave 3.3.0 vs 3.3.2 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.3.0...3.3.2`](https://github.com/remnawave/backend/compare/3.3.0...3.3.2)
and checking the matching
[`remnawave/frontend` flow](https://github.com/remnawave/frontend/compare/3.3.0...3.3.2).
The backend comparison contains 6 commits and 6 files with 16 additions and 7
deletions. Tag `3.3.2` resolves to backend commit
`347e6de129f0289a3831dfbb7452d36b49528f3c`; the previous `3.3.0` tag resolves to
`542f131487f24bcc0121ae0d1cdd58cea8fd9d86`.

Remnawave 3.3.1 and 3.3.2 are contract-compatible patches for the Terraform
provider. No REST route, request DTO, or response model changed, so no
version-specific provider logic is required. Only three source files carry
behavioral change: the node-plugin JSON schema, the node-plugin repository, and
a Telegram notification URL constant.

## Provider-relevant changes

### `torrentBlocker.rulePlacement` in the node-plugin schema

Remnawave 3.3.1 adds an optional key to `TorrentBlockerPluginSchema`:

```text
rulePlacement: number, min 0, max 1000, optional
```

It sets the position of the rule Remnawave injects into the `routing.rules`
array. Remnawave 3.3.1 declared a schema default of `0`; 3.3.2 removes that
default so an omitted key stays absent instead of being materialized in stored
plugin configuration.

`remnawave_node_plugin.plugin_config` is an opaque JSON string that the provider
normalizes rather than models field by field, so the key needs no schema change.
Both patch releases still change provider behavior, because
`TorrentBlockerPluginSchema` is a plain non-strict `z.object` and
`node-plugins.service.ts` stores the parsed output (`inputConfig =
validatedConfig.data`) rather than the request body:

- **3.3.0 silently strips the key.** It is not a validation error. The panel
  stores and returns the object without `rulePlacement`, and because Create and
  Update write the response into state, Terraform would abort with "Provider
  produced inconsistent result after apply" — an opaque provider bug report
  rather than a version message. The provider therefore gates the nested key
  with `isVersionAtLeast3_3_1` and fails with `torrentBlocker.rulePlacement
  requires Remnawave 3.3.1 or newer`.
- **3.3.1 injects `rulePlacement: 0`.** Its schema default materializes the key
  for every `torrentBlocker` config that omits it, which breaks apply and leaves
  permanent drift on refresh for configurations that were valid on 3.3.0. The
  provider drops a returned `rulePlacement` when the configuration did not set
  one (`alignNodePluginRulePlacement`), applied on create, update, and refresh.
  A configured value — including an explicit `0` — is preserved.

3.3.2 removed the default, which is why the provider must not materialize one of
its own the way it does for `sharedLists`: that would produce a permanent diff
against 3.3.2 panels that omit the key.

Note for examples and tests: `enabled`, `blockDuration`, and `ignoreLists` are
required by `TorrentBlockerPluginSchema` on every 3.3.x release, and the
`ignoreLists.ip` entries accept plain addresses or `ext:` list references, not
CIDR ranges.

### Node-plugin reorder fix

`NodePluginRepository.reorderMany` wrote `viewPosition` values to the
`externalSquads` table instead of `nodePlugin`, so reordering node plugins was
silently applied to the wrong entity on 3.3.0. Remnawave 3.3.2 fixes the target
table and the join predicate.

The provider does not expose a node-plugin reorder operation and does not manage
`viewPosition` for node plugins, so no resource attribute changes. The fix does
mean that panels reordered through the 3.3.0 web UI may have unexpected
`externalSquads.viewPosition` values; `remnawave_external_squad` does not manage
that field either, so Terraform state is unaffected.

### Telegram notification URL

The logger constant for user links changed from
`/dashboard/management/users?user=<uuid>` to `/dashboard/open/user/<uuid>`,
matching the new `open-entity` and `quick-open` pages in the 3.3.2 frontend.
This is a notification message body, not an API contract.

## Frontend changes without a backend contract

The 3.3.0...3.3.2 frontend comparison (11 commits, 36 files) adds the
`quick-open` and `open-entity` dashboard routes, a copy-entity-link button, and
a `rulePlacement` control in the node-plugin editor. Everything consumes
existing endpoints; nothing requires new provider surface.

## Published image

The Docker Hub tag resolves to a multi-platform OCI index:

```text
remnawave/backend:3.3.2
sha256:add561a4eff6616a0f01d1165e6d48cb7a710d574b4a4c0cae3fbc0cd4c34023
```

Runnable platforms inspected before the compatibility bump:

- `linux/amd64` — `sha256:3d107a391f030d3e435d8950fb025f05e5772f613246e8fefcd8e24abebfe5d6`
- `linux/arm64` — `sha256:27ed8f95ce2cfcf2208c2ae641456f19a82ed4cb9f6f24083e1f795380c03727`

The additional `unknown/unknown` manifests are OCI attestations, not runtime
platforms.

## Compatibility matrix

The default acceptance image moves from `remnawave/backend:3.3.0` to
`remnawave/backend:3.3.2`. The supported minor line remains `3.3.x`; CI retains
3.2.3, 3.1.0, 3.0.0, 2.8.1, and 2.7.4 to protect older supported API contracts.

3.3.1 is added as its own matrix entry rather than being folded into the 3.3.x
line. It is the only release that injects a `rulePlacement` default, so it is
the only place the normalization above can regress; without a dedicated job that
behavior would have no CI coverage at all.
