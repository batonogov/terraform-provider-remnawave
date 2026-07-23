# Security Policy

## Sensitive Fields

The following provider attributes and resource fields contain sensitive data:

| Resource/Attribute | Sensitive | Write-only | Notes |
| --- | --- | --- | --- |
| `provider.api_token` | Yes | No | JWT API token |
| `provider.password` | Yes | No | Admin password for login |
| `remnawave_user.trojan_password` | Yes | No | Trojan protocol password |
| `remnawave_user.ss_password` | Yes | No | Shadowsocks password |
| `remnawave_user.vless_uuid` | No | No | VLESS UUID |
| `remnawave_api_token.token` | Yes | No | JWT token (only returned on create) |
| `remnawave_keygen.pub_key` | Yes | No | Panel public key |

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
