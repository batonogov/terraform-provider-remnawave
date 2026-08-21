# Remnawave 2.8.x vs 3.0.0 — API Contract Differences

## Summary of Breaking Changes (2.8.1 → 3.0.0)

All findings derived from diffing `github.com/remnawave/backend` tags `2.8.1` vs `3.0.0`.
76 commits, 412 files changed in `src/modules/`, 276 files in `libs/contract/`.

---

## 1. Users — uuid → numeric id (BREAKING)

The most impactful change. User `uuid` field is **REMOVED** from the database and API.
Users are now identified by numeric `id` in all routes.

### Schema Change (UsersSchema)

| 2.8.x | 3.0.0 |
|---|---|
| `uuid: z.string().uuid()` | **REMOVED** |
| `id: z.number()` | `id: z.number()` (unchanged, now primary key) |
| `shortUuid: z.string()` | `shortUuid: z.string()` (unchanged) |
| `status: z.nativeEnum(USERS_STATUS).default(...)` | `status: z.enum(USERS_STATUS)` (no default) |
| `trafficLimitBytes: z.number().default(0)` | `trafficLimitBytes: z.number()` (no default) |
| `trafficLimitStrategy: z.nativeEnum(...).default(...)` | `trafficLimitStrategy: z.enum(...)` (no default) |

### Route Changes (USERS_ROUTES)

| 2.8.x | 3.0.0 |
|---|---|
| `GET_BY_UUID: (uuid) => uuid` | `GET_BY_ID: (userId) => userId` |
| `DELETE: (uuid) => uuid` | `DELETE: (userId) => userId` |
| `ACTIONS.ENABLE: (uuid) => ...` | `ACTIONS.ENABLE: (userId) => ...` |
| `ACTIONS.DISABLE: (uuid) => ...` | `ACTIONS.DISABLE: (userId) => ...` |
| `ACTIONS.RESET_TRAFFIC: (uuid) => ...` | `ACTIONS.RESET_TRAFFIC: (userId) => ...` |
| `ACTIONS.REVOKE_SUBSCRIPTION: (uuid) => ...` | `ACTIONS.REVOKE_SUBSCRIPTION: (userId) => ...` |
| — | `ACTIONS.EXTEND_EXPIRATION_DATE: (userId) => ...` (**NEW**) |
| `ACCESSIBLE_NODES: (uuid)` | `ACCESSIBLE_NODES: (userId)` |
| `SUBSCRIPTION_REQUEST_HISTORY: (uuid)` | `SUBSCRIPTION_REQUEST_HISTORY: (userId)` |
| `GET_BY.ID`, `SUBSCRIPTION_UUID`, `TELEGRAM_ID`, `EMAIL`, `TAG` | **REMOVED** (only `SHORT_UUID` and `USERNAME` remain) |

### New User Bulk Actions

- `BULK.EXTEND_EXPIRATION_DATE` — bulk extend expiration
- `BULK.UPDATE_SQUADS` — bulk update squad assignments
- `BULK.ALL.UPDATE` / `BULK.ALL.RESET_TRAFFIC` / `BULK.ALL.EXTEND_EXPIRATION_DATE`

### New User Action

- `ACTIONS.EXTEND_EXPIRATION_DATE` — extend a single user's expiration date

---

## 2. ip-control → connections module rename (BREAKING)

The entire `ip-control` module is renamed to `connections`.

| 2.8.x Route | 3.0.0 Route |
|---|---|
| `POST /api/ip-control/fetch-ips/:userUuid` | `POST /api/connections/by-user/:userId` |
| `GET /api/ip-control/fetch-ips/result/:jobId` | `GET /api/connections/by-user/:jobId` |
| `POST /api/ip-control/drop-connections` | `POST /api/connections/drop` |
| `POST /api/ip-control/fetch-users-ips/:nodeUuid` | `POST /api/connections/by-node/:uuid` |
| `GET /api/ip-control/fetch-users-ips/result/:jobId` | `GET /api/connections/by-node/:jobId` |

Note: node-based connections still use node UUID; only user-based ones switched to numeric ID.

---

## 3. Subscription Settings — fields removed (BREAKING)

Six fields removed from SubscriptionSettingsSchema:

| Removed Field | Type |
|---|---|
| `profileTitle` | `string` |
| `supportLink` | `string` |
| `profileUpdateInterval` | `number` |
| `isProfileWebpageUrlEnabled` | `boolean` |
| `happAnnounce` | `string \| null` |
| `happRouting` | `string \| null` |

These were moved to response headers configuration / external squads.

The ExternalSquadSubscriptionSettingsSchema also drops the same fields from its `.pick()`.

---

## 4. External Squad — response headers split (BREAKING)

| 2.8.x | 3.0.0 |
|---|---|
| `responseHeaders: Record<string, string> \| null` | **REMOVED** |
| — | `responseHeadersAdd: Record<string, string>` (**NEW**) |
| — | `responseHeadersRemove: string[]` (**NEW**) |

---

## 5. HWID — userUuid → userId

| 2.8.x | 3.0.0 |
|---|---|
| `GET /api/hwid/devices/:userUuid` | `GET /api/hwid/devices/:userId` |

---

## 6. Metadata — uuid → userId/nodeId

| 2.8.x | 3.0.0 |
|---|---|
| `GET /api/metadata/user/:uuid` | `GET /api/metadata/user/:userId` |
| `UPSERT /api/metadata/user/:uuid` | `UPSERT /api/metadata/user/:userId` |

---

## 7. Bandwidth Stats — route restructure

### Users

| 2.8.x | 3.0.0 |
|---|---|
| `GET /api/bandwidth-stats/users/:uuid` | `GET /api/bandwidth-stats/users/:userId` |
| `BANDWIDTH_STATS_ROUTES.LEGACY.*` | **REMOVED** |

### Nodes — new endpoint

- `GET /api/bandwidth-stats/nodes/usage` (**NEW**)

### Internal Squads — new section

- `GET /api/bandwidth-stats/internal-squads/:uuid/usage` (**NEW**)
- `GET /api/bandwidth-stats/internal-squads/:uuid/users/:userId/usage` (**NEW**)

---

## 8. System Stats — new endpoints

- `GET /api/system/stats/digest` (**NEW**)
- `GET /api/system/stats/http` (**NEW**)

---

## 9. Internal Squads — new bulk actions

- `POST /api/internal-squads/:uuid/bulk-actions/add-many-users` (**NEW**)
- `DELETE /api/internal-squads/:uuid/bulk-actions/remove-many-users` (**NEW**)

---

## 10. API Tokens — new endpoint + rename

- `GET /api/tokens/ott` (**NEW**) — one-time token
- Controller method renames: `findAll` → `getApiTokens`, `create` → `createApiToken`, `delete` → `deleteApiToken`
- Delete now returns **204 No Content** instead of 200

---

## 11. Nodes — new bulk update

- `POST /api/nodes/bulk-actions/update` (**NEW**)

---

## 12. TLS Security — new field

- `echSockopt` added to TlsSecurityOptionsSchema

---

## 13. Delete Status Code Change

Multiple DELETE endpoints now return **204 No Content** instead of **200 OK**.
The Go client must handle empty response bodies gracefully.

---

## 14. Zod v4 Migration (internal, no API impact)

- `z.string().uuid()` → `z.uuid()`
- `z.number().int()` → `z.int()`
- `z.nativeEnum(X)` → `z.enum(X)`
- `z.string().datetime()` → `z.iso.datetime()`
- `.describe(JSON.stringify(...))` → `.meta({...})`

These are internal TypeScript validation changes. The JSON wire format is unchanged.

---

## Impact Assessment for Terraform Provider

### Critical (P0) — provider breaks without these fixes

1. **User routes**: all user CRUD/actions need numeric `id` instead of `uuid` on v3.0
2. **Connections module**: `ip-control` paths → `connections` paths
3. **Subscription settings**: 6 removed fields must be Optional + v2.x-only
4. **External squad headers**: `responseHeaders` → `responseHeadersAdd` + `responseHeadersRemove`
5. **Delete 204**: client must tolerate empty response body on delete

### Important (P1) — backend additions

6. User `extend-expiration-date` action — exposed by this provider
7. System stats `digest` and `http` endpoints — not exposed by this provider
8. Bandwidth stats `internal-squads` section — not exposed by this provider
9. Node bulk update endpoint — not exposed by this provider
10. Internal squad `add-many-users` / `remove-many-users` — not exposed by this provider

### CI (P2)

11. Docker compose → `remnawave/backend:3.0.0`
12. Test matrix: add v3.0.0, keep v2.8.1 and v2.7.4
13. CLAUDE.md (formerly AGENTS.md): update compatibility section
