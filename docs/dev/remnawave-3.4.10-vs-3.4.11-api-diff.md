# Remnawave 3.4.10 vs 3.4.11 — API Contract Differences

## Summary

Review of the authoritative
[`remnawave/backend` tags `3.4.10...3.4.11`](https://github.com/remnawave/backend/compare/3.4.10...3.4.11)
(3 commits, 16 files), targeting commit
`6853666801d6d1925b6aec55824106100485777a`, found no changes to REST routes,
DTOs, response models, or node-plugin schemas. No provider runtime changes
are required.

## Security and request handling

- Protected requests with bodies authenticate before JSON parsing, preventing
  pre-authentication request-body DoS. The route guard reuses that authentication;
  bodyless requests still authenticate through the guard.
- The extracted request-auth service preserves API-token scope lookup and
  admin username/UUID validation. Login-issued JWTs still require
  `X-Remnawave-Client-Type: browser`; static API tokens do not. JWT verification
  explicitly allows only HS256.
- Public auth endpoints have a 64 KiB JSON body limit. Public subscription
  endpoints reject request bodies. Other authenticated JSON requests retain
  the 100 MB limit; compressed JSON bodies are no longer accepted.
- Proxy checks now precede body parsing. Backend-tools cookie errors are
  handled safely, JWT verification is restricted to HS256, and one-time-token
  redirects are confined to the tools path without logging the token.
- Dependencies are updated.

The provider already sends uncompressed JSON with Bearer authentication and
the required browser header for username/password auth. Its login payload is
well below 64 KiB; its subscription data source uses a protected
`/api/subscriptions` route, not the public `/api/sub` route. No configuration
changes are needed.

## Published image

Docker Hub's [3.4.11 tag metadata](https://hub.docker.com/v2/repositories/remnawave/backend/tags/3.4.11)
reports this multi-platform OCI index, pinned by Compose and `compat-versions.json`:

```text
remnawave/backend:3.4.11
sha256:a5b9f81fe2cfacec1c41bd2bb43fa1a5980228b81aa2e744be4adffa3a00135b
```

Runnable platform manifests:

- `linux/amd64`: `sha256:315807a4351345283a27215ea68210ab10650ac688b5797f0d1e0970a9f8b362`
- `linux/arm64`: `sha256:661871d839c1652ec7047b648a24429aced56fd1442b29335699d47f498dbc4e`

## Acceptance matrix policy

Test the latest patch of each supported minor line: 2.7.4, 2.8.1, 3.0.0,
3.1.0, 3.2.3, 3.3.2, and 3.4.11. Remove the extra 3.3.1 Docker target;
support for older patches is not removed. Existing unit tests
`TestAlignNodePluginRulePlacement` and `TestNodePluginRulePlacementVersion`
retain coverage of the historical injected default and version gate.
The historical compatibility reports describe the matrix at their review time.

Terraform-minor coverage remains unchanged (1.0–1.17): 7 backend targets plus
17 additional Terraform targets, for 24 acceptance jobs. The repository policy
test enforces one patch per minor line and exact alignment between required
checks and the release gate's matrix.

Changing the checked-in ruleset does not change GitHub's live settings.
Before merging, migrate live required checks to the checked-in policy:
replace the 3.4.10 primary/Terraform contexts with 3.4.11 and remove 3.3.1,
preserving other protections. No live settings are changed by this PR.

## Verification

Local unit tests pass with the race detector and 43.0% statement coverage.
Build, vet, lint, `task docs:check`, repository-security policy tests, and
release-gate, release-artifact, and release-supply-chain tests pass.
`govulncheck` finds no reachable vulnerabilities.

The complete real-panel suite passes with `task test:acc` against the pinned
3.4.11 image: 89 tests pass and 2 expected tests skip (pre-3.4 host-squads
rejection and the interactive WebAuthn passkey import placeholder).
The complete backend/Terraform matrix is left to PR CI.
