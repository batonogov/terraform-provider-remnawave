# Remnawave 3.3.2 vs 3.4.2 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.3.2...3.4.2`](https://github.com/remnawave/backend/compare/3.3.2...3.4.2).
The comparison contains 23 commits touching 180 files. Tag `3.4.2` resolves to
backend commit `989827b2e89517f334937863a71e8b572f18085c`; the previous `3.3.2`
tag resolves to `7cb3a0f247ef327c09f1bf0f661cece4fc69e23f`.

Remnawave 3.4 is a breaking contract release for the provider. Two REST
contracts changed shape (host squad exclusions and shared-list identifiers),
and the name grammar for snippets and shared lists gained slash-separated
segments. Everything else — the new entity `tags`, the node SSH terminal, and
the customizable short-UUID generation — is either a dedicated endpoint family
or environment-only configuration and does not affect existing provider
surface yet.

3.4.1 and 3.4.2 are contract-compatible patches (3.4.2 fixes duplicate keys on
concurrent HWID device registration and OpenAPI nullable-field rendering), so
the provider gates on the minor version via `isVersionAtLeast3_4`.

## Provider-relevant changes

### Host squad exclusions: `excludedInternalSquads` → `internalSquads`

The flat request array is gone from both the create/update DTOs and the
response model. Hosts now carry a structured object (commit `a980f134`,
prisma migration `20260822150859_host_exclusion_modes`):

```text
internalSquads: {
  mode:   "EXCLUDE" | "ALLOW_ONLY",   // DB default "EXCLUDE"
  squads: string[]                    // ALLOW_ONLY requires at least one
}
```

The response always includes the object (never `null`); a host created without
the key returns `{mode: "EXCLUDE", squads: []}`. The link table was renamed
from `internal_squad_host_exclusions` to `internal_squad_host_links`.

Provider mapping:

- New flat attributes `internal_squads_mode` (Optional+Computed string,
  `EXCLUDE`/`ALLOW_ONLY`) and `internal_squads` (Optional+Computed list) on
  `remnawave_host` expose both modes, gated on 3.4+ the same way `mapper` is
  gated on 3.3+ (`requireBackend3_4`). An omitted mode defaults to `EXCLUDE`,
  matching the backend column default, so `internal_squads` can be set alone.
  A nested `internal_squads = {mode, squads}` object was implemented first
  and rejected: a non-null Optional+Computed single-nested object in state
  breaks Terraform's no-op planning — every subsequent plan (refresh or not)
  showed a perpetual "1 to change" with all absent computed attributes as
  "(known after apply)". Flat string/list attributes with the same data plan
  cleanly; verified by apply + `terraform plan`/-refresh=false round trips
  against a live 3.4.2 panel.
- Validation closes three traps found in review:
  `internal_squads_mode` without `internal_squads` is rejected — the update
  would serialize `{mode, squads: []}` and the backend treats that as "clear
  every squad link", silently destroying panel-side data; an explicitly empty
  `excluded_internal_squads = []` combined with the new attributes is
  rejected too — an empty configured list would fight the read-side mirror
  forever (config `[]` vs mirrored `[uuid...]` on every plan); and
  `ALLOW_ONLY` without a squad is rejected at plan time with the provider's
  message instead of an opaque backend 400.
- The deprecated `excluded_internal_squads` attribute keeps working: on 3.4+
  the client translates it to `internalSquads {mode: "EXCLUDE", squads: ...}`
  (`adaptHostSquadsRequest`) and never sends the removed key. Reads mirror
  `EXCLUDE`-mode squads back into the old attribute so pre-3.4 configurations
  do not drift; `ALLOW_ONLY` has no legacy representation and mirrors an empty
  list. Configuring both forms at once is rejected.
- Downgrading a panel from 3.4 to 3.3 nulls `internal_squads_mode` and
  `internal_squads` in state on the next refresh, surfacing a plan diff if
  the configuration set them. That is expected.

### Shared-list identifiers moved off the URL path

Because shared-list names may now contain `/` (see below), the name no longer
round-trips as a path segment (commit `889f526d`):

| Operation | 3.3.x | 3.4+ |
| --- | --- | --- |
| Get by name | `GET /api/node-plugins/shared-lists/:name` | `GET /api/node-plugins/shared-lists/by-name?name=<name>` |
| Delete | `DELETE /api/node-plugins/shared-lists/:name` | `DELETE /api/node-plugins/shared-lists` + body `{"name"}` |

Note the dedicated `by-name` segment: hitting the bare list route with a
`?name=` query silently returns the full preview list instead of the single
entity (the query is ignored), which is why the route was verified against
the live panel.

The provider picks the route through `isVersionAtLeast3_4` and encodes the
query with `url.Values` so slashes stay percent-encoded. The DELETE body
follows the existing `DeleteSnippet`/`DeletePasskey` precedent (the HTTP
helper marshals bodies for any verb). `POST`/`PATCH`/list routes are
unchanged.

### Slash-separated snippet and shared-list names

Both name grammars changed to allow `/` between segments (commit `b3c249b9`):

```text
^[A-Za-z0-9_-]+(/[A-Za-z0-9_-]+)*$
```

The provider relaxed its local shared-list name regex to the same shape. The
validator cannot know the panel version at plan time, so it is relaxed for all
versions; pre-3.4 panels reject slash-containing names with their own 400.
Snippet names were never validated locally and needed no change.

### Connection drop without connected nodes answers 500/A219

`POST /api/connections/drop` with no connected Xray node used to fail with
404; 3.4 returns `500` with `errorCode A219` ("Connected nodes not found")
before any user lookup happens (verified against both an existing and a
nonexistent user id). The endpoint contract itself is unchanged, so only the
drop-connections acceptance test needed to widen its expected-error pattern
to `status (404|500)`.

## Backend changes without provider surface (yet)

- **Entity tags** (commit `9d6e19ab`): config profiles, subpage configs, node
  plugins, subscription templates, internal squads, and external squads gain
  `tags: string[]` (max 10, each `^[A-Z0-9_:]+$`, max 36 chars). Tags are
  returned by the entity responses but are managed through dedicated
  `GET/PATCH <controller-root>/tags` endpoints rather than create/update.
  Planned as a follow-up provider feature.
- **Node SSH** (commit `4c21875c`): a websocket terminal module with SSH
  ticket creation and OPRF-based credential vault evaluation. Runtime
  operations, not configuration surface.
- **Customizable short-UUID generation** (commit `d1d3b990`): an
  env-driven length/pattern configuration. No REST contract.
- **`isDisabled` in host updates became optional** (commit `e39f9773`). The
  provider always sends an explicit boolean, which remains valid.
- **Base64 share-link mapper coverage** (commit `06c44e63`): the host-mapper
  grammar gains `$link.address`/`$link.port`/`$link.password`/
  `$link.remark`/`$link.method` targets that rewrite the generated share link
  itself rather than its query string. Additive and opaque to the provider —
  `mapper` is an unvalidated JSON string, and the mapper acceptance test
  passes unchanged on 3.4.2.
- **default SingBox template**, **OAuth2 state / passkey challenge keying**,
  and **error-handling/openapi refactors** are internal or
  subscription-generation changes.
- 3.4.1/3.4.2 patches: version bump only, then a concurrent HWID registration
  duplicate-key fix and an OpenAPI nullable-fields rendering fix.

## Published image

The Docker Hub tag resolves to a multi-platform OCI index:

```text
remnawave/backend:3.4.2
sha256:cef199b732c365ef01c3a907f56cc019733b8b1041d4b6161f5f7904e515a9ee
```

Runnable platforms inspected before the compatibility bump:

- `linux/amd64` — `sha256:ff652c45fc5c0c6c8641f850090d64408ca488e179fb60abf182e37b9266ec26`
- `linux/arm64` — `sha256:6207d3b24f0c7ff4669fee26038cb6481ddeeae244d4106f895bd862e94fed59`

The additional `unknown/unknown` manifests are OCI attestations, not runtime
platforms.

## Verification

The complete `TestAcc*` suite from this branch passed against the 3.4.2 index
digest and — with `REMNAWAVE_VERSION=3.3.2 REMNAWAVE_DIGEST=<3.3.2 digest>` —
against the previous default, proving both sides of every 3.4 gate (host squad
translation, shared-list routes, slashed names):

```text
3.4.2  PASS  ok  github.com/batonogov/terraform-provider-remnawave/provider  63.348s  (88 passed, 1 skipped)
3.3.2  PASS  ok  github.com/batonogov/terraform-provider-remnawave/provider  62.276s  (87 passed, 3 skipped)
```

The only skip on 3.4.2 is `TestAccPasskeyResource_ImportSkip` (needs a
WebAuthn fixture) — `TestAccHostInternalSquads_Pre3_4Rejected` also skips
there, because its rejection path only exists on older panels. On 3.3.2 that
test **passes** (the provider's "requires Remnawave 3.4 or later" error), the
two 3.4-only tests skip through their `isBackendAtLeast3_4` guards, and the
drop-connections test asserts the exact legacy 404.

New acceptance coverage: `TestAccHostInternalSquads` (EXCLUDE → ALLOW_ONLY →
deprecated-attribute translation → import) and `TestAccSharedListSlashedName`
(create → data source → update → import → body-based delete).
`TestAccDropConnectionsResource_ByUserUUID` widened its expected-error pattern
to `status (404|500)` for the A219 change above.

## Compatibility matrix

The default acceptance image moves from `remnawave/backend:3.3.2` to
`remnawave/backend:3.4.2`, and `3.4.x` becomes the newest supported minor
line. CI retains 3.3.2, 3.3.1, 3.2.3, 3.1.0, 3.0.0, 2.8.1, and 2.7.4 to
protect older supported API contracts, so the matrix now covers eight
versions. `Acceptance Tests (3.4.2)` is added to the required status checks;
run `scripts/configure-repository-security.sh` after merging so the live
ruleset picks the new context up.
