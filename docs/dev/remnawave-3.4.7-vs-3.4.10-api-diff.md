# Remnawave 3.4.7 vs 3.4.10 — API Contract Differences

## Summary

Review of the authoritative
[`remnawave/backend` tags `3.4.7...3.4.10`](https://github.com/remnawave/backend/compare/3.4.7...3.4.10)
(9 commits, 20 files), targeting commit
`9a51c76af9e20536d3faca74cf1b938fee32ab3c`, found one additive host field.
Existing provider surfaces need no runtime adaptation.

The primary acceptance image moves from 3.4.7 to 3.4.10. Every older matrix
entry remains, including 3.3.1 for its unique `rulePlacement` default.
Feature minimums remain unchanged; `postStart` still requires 3.4.5+.

## Remnawave 3.4.8

- Patches `sirv`'s static-file extension regex to avoid excessive backtracking.
- Restricts PEM certificate file reads to `/var/lib/remnawave`, checking the
  resolved real path so symlinks cannot escape that directory. Existing
  panel/node certificate configurations referencing files elsewhere need
  operator attention; the provider does not read those files itself.
- Updates dependencies. No REST routes or DTOs change.

## Remnawave 3.4.9

- Adds optional `includeInSrrRules: string[]` to host create/update DTOs and
  the field to host responses. The database defaults to `[]`.
- An empty list keeps the host available to all subscription response rules;
  a non-empty list restricts it to the named matched rules.
- The provider does not expose this field yet. Unknown response properties
  are ignored, and writes omit the field rather than resetting panel-managed
  values. A client regression test covers create/read/update/list with the
  legacy response, empty list, and configured rules.
- No route changes or node-plugin schema changes.

Exposing `include_in_srr_rules` as a version-gated Terraform host attribute
is follow-up work, not part of this compatibility update.

## Remnawave 3.4.10

- Passes the matched rule name to proxy resolution so `includeInSrrRules`
  filtering works. Raw subscription rendering uses the literal rule name
  `RAW`.
- Defaults an omitted raw-subscription `withDisabledHosts` query argument to
  `false`. The provider does not call that short-UUID raw-subscription route.
- No further DTO, route, or plugin-schema changes.

## Published image

Docker Hub's [3.4.10 tag metadata](https://hub.docker.com/v2/repositories/remnawave/backend/tags/3.4.10)
reports this multi-platform OCI index:

```text
remnawave/backend:3.4.10
sha256:19efa7828b3a2e2c2f8f8503c175d84c9391358a6b1a5fd3b8112b8064ca6038
```

Runnable platform manifests:

- `linux/amd64`: `sha256:c5006a4fe0aca6e4a7851b6f7765e454d97ec0f67bd74bbea856adfc15201c85`
- `linux/arm64`: `sha256:235c57f8cfcc998d5de38d1cc12b84be053d50f45ab070c88caa2f01e4afa699`

Compose and `compat-versions.json` pin the index digest.

## Verification

Local unit tests with the race detector pass with 43.0% statement coverage
(above the 30% CI floor), including the additive host-field regression test.
Build, vet, lint, repository-security policy tests, release-gate tests, and
`task docs:check` pass.

The real-panel suite is run with `task test:acc` against the pinned image;
the full backend/Terraform matrix remains enforced by CI.

## Required checks

The checked-in main ruleset replaces the primary backend checks with
`Acceptance Tests (3.4.10)` and its Terraform-minor variants, matching the
matrix derived from `compat-versions.json`.

Changing this JSON does not update GitHub's live ruleset. Before merging,
replace the corresponding 3.4.7 checks in the live main ruleset while
preserving all other protection settings.
