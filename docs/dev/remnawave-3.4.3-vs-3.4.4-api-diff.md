# Remnawave 3.4.3 vs 3.4.4 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.4.3...3.4.4`](https://github.com/remnawave/backend/compare/3.4.3...3.4.4)
(10 commits, 27 files, 609 additions, 367 deletions). Tag `3.4.4` resolves to
backend commit `c797111d1dc24a51126899d3dc62f113b100964a`; the `3.4.3` tag
resolves to `5c88faac5785a476a83e93cf20d3bbe59bb338a8`.

Remnawave 3.4.4 is additive-only for existing surfaces: no existing REST
route, request DTO, or response model changes. The Terraform provider needs
only a compatibility-matrix bump.

## Additions

- **Clone host endpoint** (`ac57dd84`): `POST /api/hosts/actions/clone` with
  request body `{ "cloneFromUuid": <uuid> }`, scope `clone`, error code
  `A258` (`CLONE_HOST_ERROR`), and the standard host response schema. The
  backend-contract package moved `3.4.13` → `3.4.14`. The endpoint is not a
  provider surface yet and is a follow-up candidate like entity tags.
- **Next traffic reset template variables** (`937ff20b`):
  `NEXT_TRAFFIC_RESET_AT`, `NEXT_TRAFFIC_RESET_AT_UNIX`, and
  `LAST_TRAFFIC_RESET_AT` join `TEMPLATE_KEYS`; the date variants accept a
  `format` argument (dayjs syntax, default `DD.MM.YYYY`). The
  `subscription-refill-date` header now comes from the same
  `getNextTrafficResetAt` util, which adds `MONTH_ROLLING` support and small
  fixed offsets (00:05 daily, 00:15 weekly, 00:20 monthly). Templates and
  headers are opaque to the provider.
- **Seeded default response rules** (`7b3c2fbe`): the default SRR config
  user-agent regex adds the `rabbit` client. Only fresh panels pick this up;
  the provider treats response rules as opaque JSON either way.

## Fixes

- **Node name trimming on update** (`ce86b492`): the update path now applies
  `name.trim()` (the `address` field was already trimmed). A configured node
  `name` with surrounding whitespace will read back trimmed and drift until
  the configuration trims it. The same commit also restarts an enabled node
  only when connection-relevant fields actually change — an internal
  optimization with no contract impact.
- **Subscription request stream typo** (`44e87e35`): the queue processor now
  writes `srrResponseType` (previously misspelled `ssrResponseType`), so
  3.4.4 history records carry the field the query side expects. The
  `remnawave_subscription_request_history` data source passes records
  through as raw JSON and needs no change.

## Chores

BullMQ job retention/count limits for completed and failed jobs
(`74e27b7e`), dependency updates, and a base-image move to
`node:24.21-trixie-slim` — no API impact.

## Published image

Docker Hub resolves to the following multi-platform OCI index:

```text
remnawave/backend:3.4.4
sha256:63ef481550bbf49dabfa514c95d94109619cc85607730b308f7ad0b9b5599f06
```

Runnable platforms inspected before the compatibility bump:

- `linux/amd64` — `sha256:1464eea7ad54795b50fc9841b242310154c75727dd78bfc9fe70f89a35f9a8ab`
- `linux/arm64` — `sha256:41bd4438063d5cdf13fe27eb94e95bdbd494dace560b612a08776a5a5dcee03a`

The additional `unknown/unknown` manifests are OCI attestations, not runtime
platforms.

## Verification

The complete `TestAcc*` suite passed against the exact 3.4.4 index digest:

```text
PASS
ok  github.com/batonogov/terraform-provider-remnawave/provider  65.569s
```

The import-only WebAuthn passkey test retained its expected skip because no
pre-existing passkey fixture was supplied, and the pre-3.4 host-squads
rejection test skipped because the default panel is now 3.4.4.

## Compatibility matrix

The default acceptance image moves from `remnawave/backend:3.4.3` to
`remnawave/backend:3.4.4`. The supported minor line remains `3.4.x`; CI
retains 3.3.2, 3.3.1, 3.2.3, 3.1.0, 3.0.0, 2.8.1, and 2.7.4 to protect older
supported API contracts.
