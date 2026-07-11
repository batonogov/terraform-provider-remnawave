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

## Reporting

Report security issues privately to the repository owner.
