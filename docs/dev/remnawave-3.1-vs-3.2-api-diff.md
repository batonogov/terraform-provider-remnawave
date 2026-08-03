# Remnawave 3.1.0 vs 3.2.0 — API Contract Differences

## Summary

These findings come from diffing `github.com/remnawave/backend` tags `3.1.0`
and `3.2.0` (3 commits, 16 files). The release is **additive** for the
Terraform provider: every existing route, request payload, and response
contract is unchanged. No version-specific provider logic is required.

## 1. New endpoint — `GET /api/system/configuration`

A read-only endpoint (scope `configuration`, kind `read`) that returns a
snapshot of the panel's server-side environment configuration. It does **not**
manage state — the values come from the panel's env/config via
`getOrThrow`/`getIfEnabled`, so it is a data source candidate, not a resource.

Response shape (`{ "response": { ... } }` envelope):

```
notifications:
  webhook:                 boolean   # WEBHOOK_ENABLED
  bandwidthUsage:          number[] | null   # BANDWIDTH_USAGE_NOTIFICATIONS_THRESHOLD
  notConnectedAfter:       number[] | null   # NOT_CONNECTED_USERS_NOTIFICATIONS_AFTER_HOURS
  expirationNotifications: number[] | null   # EXPIRATION_NOTIFICATIONS
service:
  cleanUsageHistory:        boolean   # SERVICE_CLEAN_USAGE_HISTORY
  disableUserUsageRecords:  boolean   # SERVICE_DISABLE_USER_USAGE_RECORDS
  disableSrhRecords:        boolean   # SERVICE_DISABLE_SRH_RECORDS
  exportToRedisStream:      boolean   # EXPORT_TO_STREAM_ENABLED
misc:
  shortUuidLength:          number    # SHORT_UUID_LENGTH
  userUsageIgnoreBelowBytes: number   # USER_USAGE_IGNORE_BELOW_BYTES
  subPublicDomain:          string    # SUB_PUBLIC_DOMAIN
```

Provider impact: **none** in this release. The endpoint is tracked as a
follow-up candidate for a future `remnawave_system_configuration` data source.

## 2. Config-profile outbound validation — refactored, not restricted

The `XRayConfig` validator extracts the existing "config must have outbounds"
check out of the combined `validate()` method into a dedicated
`validateOutbounds()`, and `ConfigProfileService` calls it explicitly on the
config-profile update path. The check itself and the error message are
identical to 3.1.0; behaviour on the update path is equivalent. No provider or
acceptance fixture change is required.

## Other changes

- `chore: bump version to 3.2.0` (package.json, lockfile, contract package).

## Compatibility matrix

The default acceptance image moves to `remnawave/backend:3.2.0`. CI retains
3.1.0, 3.0.0, 2.8.1, and 2.7.4 entries to ensure the additive endpoint and the
config-profile validation refactor do not regress older supported minor lines.
