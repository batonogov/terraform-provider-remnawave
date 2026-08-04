# Remnawave 3.2.0 vs 3.2.1 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.2.0...3.2.1`](https://github.com/remnawave/backend/compare/3.2.0...3.2.1)
(4 commits, 9 files). Remnawave 3.2.1 is a contract-compatible patch for the
Terraform provider: no REST route, request DTO, or response model changed, so
no version-specific provider logic is required.

## Provider-relevant changes

### Invalid API-token UUID cleanup

The startup scope migration now validates existing API-token UUIDs and deletes
legacy rows whose UUID is invalid before migrating their scopes. API tokens
created through the supported Remnawave API already receive valid UUIDs, so the
provider's `remnawave_api_token` request and response contracts are unchanged.

This may remove an already-invalid legacy token during a panel upgrade, but it
does not introduce a new API shape or a Terraform migration requirement.

### Subscription generation internals

The template engine was refactored to parse and cache template variables and to
support optional labels for status and reset-strategy values. The Mihomo
generator now carries custom WebSocket and HTTP Upgrade headers into generated
client configuration. These paths generate subscription output and do not
change resources or data sources exposed by the provider.

### Dependency cleanup

The backend removed the unused `transliteration` package and bumped its package
metadata from 3.2.0 to 3.2.1. This does not affect the provider API contract.

## Published image

The Docker Hub tag resolves to a multi-platform OCI index:

```text
remnawave/backend:3.2.1
sha256:a6302ff950e1946a70c76567fe323dcd9a4f5f35d563510cf50ca4f64a52921f
```

Runnable platforms inspected before the compatibility bump:

- `linux/amd64` — `sha256:017222725b7e018a8eb4d30ddbafbefeef07868bd4cbb4751c02fbfed2da6d22`
- `linux/arm64` — `sha256:aed32b8d383906241a778c8a92b631d21c62297d8055181243d66758cec4e04a`

The additional `unknown/unknown` manifests are OCI attestations, not runtime
platforms.

## Verification

The complete `TestAcc*` suite from provider `main` commit
`7be113264c8b0d35525df15cf4fb2443b72b2179` passed against the exact 3.2.1
index digest:

```text
PASS
ok  github.com/batonogov/terraform-provider-remnawave/provider  118.138s
```

## Compatibility matrix

The default acceptance image moves from `remnawave/backend:3.2.0` to
`remnawave/backend:3.2.1`. The supported minor line remains `3.2.x`; CI retains
3.1.0, 3.0.0, 2.8.1, and 2.7.4 to protect older supported API contracts.
