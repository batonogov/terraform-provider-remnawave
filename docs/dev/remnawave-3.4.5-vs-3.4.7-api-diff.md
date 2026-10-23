# Remnawave 3.4.5 vs 3.4.7 — API Contract Differences

## Summary

Review of the authoritative
[`remnawave/backend` tags `3.4.5...3.4.7`](https://github.com/remnawave/backend/compare/3.4.5...3.4.7)
(5 commits, 9 files) found no changes to REST routes, request DTOs, response
models, contracts, or node-plugin schemas. The target backend commit is
`52321390ac` (tag `3.4.7`). No provider runtime changes are required.

Both patches were released on October 22, 2026. This update replaces the
primary 3.4.5 acceptance image with 3.4.7 and retains every older matrix entry,
including 3.3.1 for its unique `torrentBlocker.rulePlacement` default.
Feature minimums remain unchanged: in particular, `postStart` still requires
3.4.5+, rather than 3.4.7+.

## Remnawave 3.4.6

- [`b5551e97`](https://github.com/remnawave/backend/commit/b5551e97)
  replaces the raw-SQL bulk-delete-by-status builder with a Kysely query.
  It preserves the status predicate, default limit of 30,000, and numeric
  deletion count. The provider's `remnawave_user_bulk_action` deletes explicit
  user identifiers; it does not expose deletion by status.
- [`e832e63c`](https://github.com/remnawave/backend/commit/e832e63c)
  updates the Docker base image from `node:24.21-trixie-slim` to
  `node:24.22-trixie-slim` for the Node.js 24.22.1 security release. The
  dependency lockfile also changes; no provider dependency changes are needed.

## Remnawave 3.4.7

[`6863f619`](https://github.com/remnawave/backend/commit/6863f619)
fixes X25519 public-key derivation for REALITY and VLESS encryption. The
backend derives a base64url public key from an existing raw private key with
`@noble/curves` instead of constructing a Node.js JWK with an empty public
coordinate. REALITY key resolution reuses the helper and its existing cache.

This affects generated subscriptions, not the provider's
`remnawave_keygen` data source, which reads the panel public key for node setup
from `GET /api/keygen`. The provider passes config-profile JSON through to the
backend and needs no cryptographic implementation change. The acceptance
suite verifies the provider contract; it does not specifically reproduce the
upstream X25519 regression.

## Published image

Docker Hub's [3.4.7 tag metadata](https://hub.docker.com/v2/repositories/remnawave/backend/tags/3.4.7)
reports this multi-platform OCI index:

```text
remnawave/backend:3.4.7
sha256:5c26c0314be896dfac82020b693c50a73d2f18ef593da2ef97244330ac032bcf
```

Runnable platform manifests:

- `linux/amd64`: `sha256:98af5f91aabb2247be8340ad3fb8f88de831d09bd92b8e77bd9ed02d9843af4a`
- `linux/arm64`: `sha256:d34541e5a7abdc0a401e9db49297c0bced5c104942530106020a0cf8c0c80fb3`

Compose and `compat-versions.json` pin the index digest, so local and CI
acceptance runs use the same published image.

## Verification

The complete `TestAcc*` suite passed against the pinned 3.4.7 index digest:
89 tests passed and 2 expected tests skipped (the pre-3.4 host-squads rejection
test and the import-only WebAuthn passkey test without an external fixture).

Local unit tests with the race detector passed with 42.5% statement coverage
(above the 30% CI floor). Build, vet, lint, repository-security policy tests,
and release CI verification tests also passed. Terraform example validation,
documentation generation/validation, inventory checks, and the final
clean-documentation check (`task docs:check`) passed.

Two rounds of independent agent reviews checked upstream compatibility and
the final matrix/digest/ruleset/documentation diff. No functional findings
remained; the second round clarified the completed documentation checks.

## Required checks

The checked-in main ruleset requires `Acceptance Tests (3.4.7)` in place of
`Acceptance Tests (3.4.5)`, matching the matrix derived from
`compat-versions.json`. The live GitHub main ruleset was also updated to
require `Acceptance Tests (3.4.7)`, preserving its other protection settings.
Changing the checked-in JSON alone does not update GitHub settings.
