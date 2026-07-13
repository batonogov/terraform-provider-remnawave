# Remnawave API coverage

This document tracks provider coverage against the Remnawave backend contracts.
It distinguishes declarative Terraform state from read-only queries and
imperative operations; counting every REST route as a resource would produce an
unsafe and misleading provider design.

## Audited baseline

- Backend release: `2.8.0`
- Backend source commit: `798d74986db5984364897464306928973bce3b67`
- Acceptance image: `remnawave/backend:2.8.0@sha256:cbc6d2ea0a84d8414dec565bb9ce299a3318b9e5297496baf8efcff2a1e66c65`
- Contract inventory: 184 `*.command.ts` files under `libs/contract/commands`
- Provider surface: 19 resources, 20 data sources, and 88 exported client operations

The 184 backend commands and 88 client operations are intentionally different
metrics. Backend commands include authentication, passkeys, public subscription
delivery, bulk mutations, reordering, streaming, compatibility endpoints, and
one-shot tools that are not all suitable for Terraform state.

## Current coverage

| API family | Declarative/read coverage | Remaining backend surface |
| --- | --- | --- |
| Users | CRUD resource, list data source, subscription lookup, connection keys, metadata, HWID | Selectors, tags, accessible nodes, stream, per-user history, enable/disable/reset/revoke, bulk operations |
| Nodes | Full create/update payload, computed runtime/system/version state, list/metrics data sources, metadata | Tags, enable/disable/restart/reset/reorder, bulk operations |
| Hosts | Full create/update payload including transport JSON, TLS verification, Mihomo, template and exclusion fields; list data source | Tags, reorder and bulk operations |
| Config profiles | CRUD, computed inbounds/nodes, list data source | Computed-config and standalone inbound queries, reorder |
| Internal squads | CRUD and computed accessible nodes | List data source, reorder and bulk membership operations |
| External squads | CRUD including templates, subscription/HWID settings, remarks, headers, host overrides and subpage | List data source, reorder and bulk membership operations |
| Subscription settings | Singleton resource including remarks, response headers/rules and HWID settings | Covered for the 2.8 contract |
| Subscription templates | CRUD including type and template body | List data source and reorder |
| Subscription page configs | CRUD | List data source, clone and reorder |
| Node plugins | CRUD with the 2.8 plugin configuration document | List data source, clone/reorder, executor, torrent reports and report truncation |
| Snippets | CRUD | Covered for the 2.8 contract |
| API tokens | Create, existence read through list, delete, expiry and scopes | Scopes discovery data source |
| Panel settings | Singleton resource | Covered for the 2.8 contract; requires administrator JWT |
| Infrastructure billing | Provider, billing-node and billing-history resources; billing-node/history reads | Provider list data source |
| Metadata | User and node metadata resources | Covered for the 2.8 contract |
| HWID | Device resource, user-device read path, aggregate statistics and top-users data sources | All-devices query and delete-all action |
| Bandwidth/system/history | Health, stats, recap, node metrics, bandwidth, realtime and request-history data sources | System metadata, detailed node/user bandwidth variants, legacy endpoints |
| Key generation | Public-key data source | X25519 generation and SRR matcher tools |
| IP control | None | Fetch/result polling and drop-connections operations |
| Authentication/passkeys | Provider login and JWT refresh only | Registration, OAuth, passkey and session-management APIs are deliberately outside infrastructure state |
| Public subscription delivery | Administrative subscription lookups | Raw subscription and rendered subpage delivery are application-facing endpoints |

## Coverage guarantees

The following checks are required for supported functionality:

1. Every exported `Client` method has an `httptest` contract case that checks
   its HTTP method, path, query, request body, authentication and response
   envelope handling.
2. Every registered resource and data source has a real-panel acceptance test
   against the pinned Remnawave 2.8.0 image.
3. Stateful resources test create/read/delete. Mutable resources also test an
   update, and import is tested where the backend exposes a stable identifier.
4. Backend-normalized JSON is tested with contract-valid payloads so Terraform
   does not produce an inconsistent state after apply.
5. The unit suite runs with the race detector and must stay above the CI 30%
   statement-coverage floor.

At the audited baseline there are 42 acceptance tests. The complete suite runs
with username/password administrator authentication; the API-token matrix skips
only the two administrator-only surfaces (`remnawave_api_token` and
`remnawave_panel_settings`).

## Expansion policy

Remaining functionality should be added in this order:

1. Read-only list and selector endpoints as data sources.
2. One-shot mutations as Terraform Actions where the framework and Terraform
   version support them: reset, restart, revoke, enable/disable, reorder, clone,
   bulk membership, IP control and plugin executor operations.
3. Report/history endpoints as data sources, with pagination and filters
   represented explicitly in the schema.

Authentication registration, OAuth callbacks, passkeys, streaming responses,
and public subscription-rendering endpoints should not become resources. They
belong to login/session or application-delivery workflows and cannot be made
idempotent by Terraform state.

When the compatibility target changes, re-run the contract inventory, update
this matrix and `compat-versions.json`, then run the full acceptance suite
against an explicitly pinned tag and digest.
