# Contributing

## Development Setup

```bash
# Install Go 1.26+
# Install Docker (for acceptance tests)

# Clone
git clone https://github.com/batonogov/terraform-provider-remnawave.git
cd terraform-provider-remnawave

# Build
go build -o terraform-provider-remnawave

# Run unit tests
go test ./provider -skip '^TestAcc' -count=1 -v

# Run acceptance tests (requires Docker)
docker compose up -d --wait
# Register admin (first run only)
curl -sf -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Forwarded-For: 127.0.0.1" \
  -H "X-Forwarded-Proto: https" \
  -d '{"username":"admin","password":"TestAdminPassword1234567"}'
# Create API token
ADMIN_JWT=$(curl -sf -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Forwarded-For: 127.0.0.1" \
  -H "X-Forwarded-Proto: https" \
  -d '{"username":"admin","password":"TestAdminPassword1234567"}' | jq -r '.response.accessToken')
API_TOKEN=$(curl -sf -X POST http://localhost:3000/api/tokens \
  -H "Content-Type: application/json" \
  -H "X-Forwarded-For: 127.0.0.1" \
  -H "X-Forwarded-Proto: https" \
  -H "X-Remnawave-Client-Type: browser" \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -d '{"name":"test","expiresInDays":1,"scopes":["*"]}' | jq -r '.response.token')
# Run tests
TF_ACC=1 REMNAWAVE_ENDPOINT=http://localhost:3000 \
  REMNAWAVE_API_TOKEN=$API_TOKEN REMNAWAVE_PROXY_HEADERS=true \
  go test ./provider -run TestAcc -count=1 -timeout 120s -v
```

## PR Workflow

1. Create a branch from `main`
2. Make changes, add tests
3. Ensure `go vet ./...` and `go build ./...` pass
4. Ensure CI is green (lint + build + unit + acceptance)
5. Create PR with conventional commit messages (`feat:`, `fix:`, `docs:`, etc.)
6. Squash merge after approval

## Conventions

- **Commits**: Conventional Commits (`feat:`, `fix:`, `docs:`, `ci:`, `test:`, `chore:`)
- **File naming**: `provider/resource_<name>.go`, `provider/data_source_<name>.go`
- **Tests**: `TestAcc<Resource>` for acceptance, `Test<Unit>` for unit
- **Linting**: golangci-lint with `.golangci.yml` config
