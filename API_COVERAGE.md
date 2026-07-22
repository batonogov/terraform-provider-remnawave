# Remnawave API coverage

This document tracks provider coverage against the Remnawave backend contracts.
It distinguishes declarative Terraform state from read-only queries and
imperative operations; counting every REST route as a resource would produce an
unsafe and misleading provider design.

## Audited baseline

- Backend release: `2.8.1`
- Backend source commit: `ba51868149362d0b9ac0e23133d0532176ccb5a2`
- Acceptance image: `remnawave/backend:2.8.1@sha256:361f9bb0b183d4fcefea2f1f7163db490e2aa1ec3b4bdde016a9ab9229ce956b`
- Contract inventory: 184 `*.command.ts` files under `libs/contract/commands`
- Provider surface: 26 resources, 23 data sources, and 105 exported client operations

The 184 backend commands and 105 client operations are intentionally different
metrics. Backend commands include authentication, passkeys, public subscription
delivery, bulk mutations, reordering, streaming, compatibility endpoints, and
one-shot tools that are not all suitable for Terraform state.

## Current coverage

| API family | Declarative/read coverage | Remaining backend surface |
| --- | --- | --- |
| Users | CRUD resource, list data source, subscription lookup, connection keys, metadata, HWID, single-user actions and bulk actions | Selectors, tags, accessible nodes, stream and detailed per-user history |
| Nodes | Full create/update payload, computed runtime/system/version state, list/metrics data sources, metadata, single-node actions and bulk actions | Tags and reorder operations |
| Hosts | Full create/update payload including transport JSON, TLS verification, Mihomo, template and exclusion fields; list/tags data sources and bulk actions | Reorder operations |
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
| IP control | Asynchronous user-IP lookup data source and drop-connections action resource | Lower-level fetch/result jobs are encapsulated by the data source |
| Authentication/passkeys | Provider login/automatic re-authentication, passkey list data source, and import-only passkey read/delete resource | Registration, OAuth and session-management APIs are deliberately outside infrastructure state |
| Public subscription delivery | Administrative subscription lookups | Raw subscription and rendered subpage delivery are application-facing endpoints |

## Coverage guarantees

The following checks are required for supported functionality:

1. Every exported `Client` method has an `httptest` contract case that checks
   its HTTP method, path, query, request body, authentication and response
   envelope handling.
2. Every registered resource and data source that can be exercised
   non-interactively has real-panel acceptance coverage against both pinned
   Remnawave 2.8.1 and 2.7.4 images. The import-only passkey resource is the
   explicit exception because creating its fixture requires a WebAuthn ceremony.
3. Declarative resources exercise representative lifecycle paths; this matrix
   does not imply that every mutable resource has every update/import permutation.
   Imperative resources use non-destructive actions or assert the expected
   backend diagnostic when the fixture cannot succeed.
4. Backend-normalized JSON is tested with contract-valid payloads so Terraform
   does not produce an inconsistent state after apply.
5. The unit suite runs with the race detector and must stay above the CI 30%
   statement-coverage floor.

At the audited baseline there are 68 `TestAcc` entry points. The compatibility
matrix first runs the suite with API-token authentication, then reruns the three
administrator-only checks (`remnawave_api_token`, `remnawave_panel_settings`,
and `remnawave_passkeys`) with username/password authentication. Only the
interactive passkey resource import placeholder is permanently skipped.

## Expansion policy

Remaining functionality should be added in this order:

1. Read-only list and selector endpoints as data sources.
2. One-shot mutations as Terraform Actions where the framework and Terraform
   version support them: reorder, clone, bulk membership and plugin executor
   operations. Existing imperative resources remain for compatibility.
3. Report/history endpoints as data sources, with pagination and filters
   represented explicitly in the schema.

Authentication registration, OAuth callbacks, passkey registration, streaming
responses, and public subscription-rendering endpoints should not become
declarative resources. They belong to login/session or application-delivery
workflows and cannot be made idempotent by Terraform state.

When the compatibility target changes, re-run the contract inventory, update
this matrix and `compat-versions.json`, then run the full acceptance suite
against an explicitly pinned tag and digest.
