# Security Policy

## Sensitive Fields

The following provider attributes and resource fields contain sensitive data:

| Resource/Attribute | Sensitive | Write-only | Notes |
| --- | --- | --- | --- |
| `provider.api_token` | Yes | No | JWT API token |
| `provider.password` | Yes | No | Admin password for login |
| `provider.custom_headers` | Yes | No | May contain gateway credentials |
| `remnawave_user.short_uuid` | Yes | No | Capability used in subscription URLs |
| `remnawave_user.trojan_password` | Yes | No | Trojan protocol password |
| `remnawave_user.ss_password` | Yes | No | Shadowsocks password |
| `remnawave_user.vless_uuid` | Yes | No | VLESS client credential |
| `remnawave_user.subscription_url` | Yes | No | Subscription access URL |
| `remnawave_api_token.token` | Yes | No | JWT token (only returned on create) |
| `remnawave_keygen.pub_key` | Yes | No | Panel public key |
| `remnawave_subscriptions.short_uuid` | Yes | No | Capability used to select a subscription |
| `remnawave_subscriptions.response` | Yes | No | May contain subscription and configuration URLs |
| `remnawave_connection_keys.response` | Yes | No | Contains connection credentials |

Terraform's `Sensitive` flag redacts values from normal CLI presentation, but
does not encrypt or remove them from state. Store state in a protected backend,
restrict access to state snapshots and backups, and avoid publishing state as a
CI artifact.

## Vulnerability Gate

CI runs the pinned `govulncheck` source scan and blocks changes when provider
code can reach a known vulnerable symbol. Run the same check locally with
`task test:vuln`.

A vulnerability exception is allowed only when no fixed dependency is
available and the advisory is not practically reachable in this provider. The
exception must be reviewed in a pull request and identify the advisory, explain
the reachability analysis and compensating controls, name an owner, link a
tracking issue, and include an expiry date no more than 30 days away. CI must
limit the exception to that exact advisory and fail after the expiry date;
blanket `continue-on-error` or disabled scans are not acceptable.

## Reporting

Report security issues privately to the repository owner.

## Repository controls

Branch, tag, Actions, and release-Environment settings are versioned under
`.github/repository-settings/` and audited with
`task repo:security:check`. The rollout and narrow emergency-bypass procedure
are documented in `docs/repository-security.md`.

## Release provenance

Releases include a per-archive SPDX SBOM, GPG-signed checksums, and a
GitHub/Sigstore provenance bundle. The release remains a draft until every
archive/SBOM pair and attestation has been verified. Consumer verification
steps are documented in `docs/release-verification.md`.
