# Remnawave 3.0.0 — multi-platform image pin

The compatibility manifest previously pinned the `linux/amd64` child manifest
of `remnawave/backend:3.0.0`. This worked on GitHub's amd64 runners but selected
the wrong architecture for local arm64 compatibility checks.

Pin the published multi-platform OCI index instead:

```text
remnawave/backend:3.0.0
sha256:ac871579de421c0a36b5903bc789214c4b24d68f100d4fb3cdb4689a97821356
```

Its runnable child manifests are:

| Platform | Digest |
| --- | --- |
| `linux/amd64` | `sha256:4db37ad505101672fdba8ba39f4744817b7cf3d0c7de2e5cd6c145af2117b69c` |
| `linux/arm64` | `sha256:c44048a9c1115e5911efaad7e3b3153e1e0192d4e77db4a63d747c4a62e28e1e` |

The amd64 child is identical to the previous pin, so CI runs the same image.
Docker now selects the native arm64 child on arm64 machines. Supported versions,
API contracts, required-check names, and the acceptance matrix are unchanged.

Verified against [Docker Hub tag metadata](https://hub.docker.com/v2/repositories/remnawave/backend/tags/3.0.0)
and Docker Registry v2: the registry's `Docker-Content-Digest` and the SHA-256
computed over the raw index bytes both match the new pin. The index contains
the previous amd64 digest and the published arm64 digest.
