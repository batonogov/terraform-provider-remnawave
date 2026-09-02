# Remnawave 3.4.2 vs 3.4.3 — API Contract Differences

## Summary

These findings come from diffing the authoritative
[`remnawave/backend` tags `3.4.2...3.4.3`](https://github.com/remnawave/backend/compare/3.4.2...3.4.3)
(2 commits, 3 files, 9 additions, 10 deletions). Tag `3.4.3` resolves to backend
commit `f8ad8ad3410252215ca7b2e429d157bd275ec564`; the previous `3.4.2` tag
resolves to `fca29680290f6f010bae1100ff0f5a14ad4a3196`.

Remnawave 3.4.3 is a security patch with no REST contract change: no route,
request DTO, response model, or scope is added, removed, or altered. The
Terraform provider therefore needs only a compatibility-matrix bump.

## The fix

`src/main.ts` closes an authentication bypass on the backend-tools surface.
Previously the tools auth middleware was registered globally and matched the
request with a case-sensitive `req.path.startsWith('/api/backend-tools')`
check, so a mixed-case path such as `/API/backend-tools/...` skipped the
authentication middleware. The middleware is now mounted directly on the
backend-tools mount path — `app.use(backendToolsPath, toolsAuthMiddleware(...))`
— and the helmet exemption for the same surface uses a case-insensitive
comparison. `package.json` and `package-lock.json` carry only the version bump.

The backend-tools surface is not part of the provider surface; every route the
provider calls is an `/api/*` REST route guarded by the unchanged JWT/API-token
middleware chain. No provider request, response model, or version gate is
affected.

## Published image

Docker Hub resolves to the following multi-platform OCI index:

```text
remnawave/backend:3.4.3
sha256:4ea85b2fc16bd3e5d367b61afc07ec219133eaa12dd7b5e898adc33c84515422
```

Runnable platforms inspected before the compatibility bump:

- `linux/amd64` — `sha256:f471279e06fd02c48b18b6be49233f66a99c48e82ad512d1c99eb6e5d120e333`
- `linux/arm64` — `sha256:498a26f610d02969caf8d0bb3a8cb1c31a88ca03d5f218f34d3e1487ce2cce1b`

The additional `unknown/unknown` manifests are OCI attestations, not runtime
platforms.

## Verification

The complete `TestAcc*` suite passed against the exact 3.4.3 index digest:

```text
PASS
ok  github.com/batonogov/terraform-provider-remnawave/provider  62.504s
```

The import-only WebAuthn passkey test retained its expected skip because no
pre-existing passkey fixture was supplied.

## Compatibility matrix

The default acceptance image moves from `remnawave/backend:3.4.2` to
`remnawave/backend:3.4.3`. The supported minor line remains `3.4.x`; CI retains
3.3.2, 3.3.1, 3.2.3, 3.1.0, 3.0.0, 2.8.1, and 2.7.4 to protect older supported
API contracts.
