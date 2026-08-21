# Remnawave API coverage

This document tracks provider coverage against the Remnawave backend contracts.
It distinguishes declarative Terraform state from read-only queries and
imperative operations; counting every REST route as a resource would produce an
unsafe and misleading provider design.

## Audited backend baseline

- Backend release: `2.8.1`
- Backend source commit: `ba51868149362d0b9ac0e23133d0532176ccb5a2`
- Acceptance image: `remnawave/backend:2.8.1@sha256:361f9bb0b183d4fcefea2f1f7163db490e2aa1ec3b4bdde016a9ab9229ce956b`
- Contract inventory: 184 `*.command.ts` files under `libs/contract/commands`

This backend inventory is a historical contract snapshot. The provider now
supports Remnawave 2.7.x, 2.8.x, 3.0.x, 3.1.x, 3.2.x, and 3.3.x and adapts
version-specific contracts at runtime.

## Current provider surface

- Resources: 28
- Data sources: 27
- Exported client operations: 120
- Acceptance test entry points: 86

The historical backend command count and current client operation count above
are intentionally different metrics. Backend commands include authentication,
passkeys, public subscription delivery, bulk mutations, reordering, streaming,
compatibility endpoints, and one-shot tools that are not all suitable for
Terraform state.

## Current coverage

| API family | Declarative/read coverage | Remaining backend surface |
| --- | --- | --- |
| Users | CRUD resource, list data source, subscription lookup, connection keys, metadata, HWID, single-user actions and bulk actions | Selectors, tags, accessible nodes, stream and detailed per-user history |
| Nodes | Full create/update payload including 3.3+ integration assignments, computed runtime/system/version state, list/metrics data sources, metadata, single-node actions and bulk actions | Tags and reorder operations; 3.3+ geocheck jobs |
| Hosts | Full create/update payload including transport JSON, TLS verification, Mihomo, template and exclusion fields, and 3.3+ host mappers; list/tags data sources and bulk actions | Reorder operations |
| Config profiles | CRUD, computed inbounds/nodes, list data source | Computed-config and standalone inbound queries, reorder |
| Internal squads | CRUD, list data source and computed accessible nodes | Reorder and bulk membership operations |
| External squads | CRUD and list data source including templates, subscription/HWID settings, remarks, versioned headers, host overrides and subpage | Reorder and bulk membership operations |
| Subscription settings | Version-adaptive singleton resource including remarks, response headers/rules and HWID settings | Covered for supported contracts |
| Subscription templates | CRUD including type and template body | List data source and reorder |
| Subscription page configs | CRUD | List data source, clone and reorder |
| Node plugins | CRUD with the 2.x plugin configuration document, the 3.1+ pre-start stage, and version-aware 3.3+ global shared-list handling | Plugin list data source, clone/reorder/sync, executor, torrent reports and report truncation |
| Node integrations | 3.3+ CRUD resource and list data source, including optional node restart on update; node assignment through `remnawave_node` | Covered for supported contracts |
| Shared lists | 3.3+ global IP/CIDR and ASN list CRUD resource and preview-list data source | Explicit shared-list synchronization action |
| Snippets | CRUD plus opt-in node synchronization after update/delete on 3.2.3+ | Covered for supported contracts |
| API tokens | Create, existence read through list, delete, expiry and scopes | Scopes discovery data source |
| Panel settings | Singleton resource | Covered for supported contracts; requires administrator JWT |
| Infrastructure billing | Provider, billing-node and billing-history resources; billing-node/history reads | Provider list data source |
| Metadata | Version-adaptive user and node metadata resources | Covered for supported contracts |
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
   non-interactively has real-panel acceptance coverage against the pinned
   Remnawave 3.3.2, 3.3.1, 3.2.3, 3.1.0, 3.0.0, 2.8.1, and 2.7.4 images. The import-only
   passkey
   resource is the explicit exception because creating its fixture requires a
   WebAuthn ceremony.
3. Declarative resources exercise representative lifecycle paths; this matrix
   does not imply that every mutable resource has every update/import permutation.
   Imperative resources use non-destructive actions or assert the expected
   backend diagnostic when the fixture cannot succeed.
4. Backend-normalized JSON is tested with contract-valid payloads so Terraform
   does not produce an inconsistent state after apply.
5. The unit suite runs with the race detector and must stay above the CI 30%
   statement-coverage floor.

Each compatibility-matrix entry runs the `TestAcc` entry points counted above
with API-token authentication, then reruns the three administrator-only checks
(`remnawave_api_token`, `remnawave_panel_settings`, and `remnawave_passkeys`)
with username/password authentication. Only the interactive passkey resource
import placeholder is permanently skipped.

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
