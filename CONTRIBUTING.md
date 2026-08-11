# Contributing

## Development Setup

```bash
# Install Go 1.26.5+, Terraform 1.12+, Task, and Docker

# Clone
git clone https://github.com/batonogov/terraform-provider-remnawave.git
cd terraform-provider-remnawave

# Build
go build -o terraform-provider-remnawave

# Run unit tests
task test:unit

# Run race detection and create a coverage report
task test:coverage

# Regenerate or verify Terraform Registry documentation
task docs
task docs:check

# Run acceptance tests; the task manages the complete Docker lifecycle
task test:acc

# Test a different explicitly pinned backend build
REMNAWAVE_VERSION=3.2.3 REMNAWAVE_DIGEST=sha256:<digest> task test:acc
```

## PR Workflow

1. Create a branch from `main`
2. Make changes, add tests
3. Run `task pre-commit`
4. Ensure CI is green (lint + build + unit + docs + acceptance)
5. Create PR with conventional commit messages (`feat:`, `fix:`, `docs:`, etc.)
6. Squash merge after approval

## Conventions

- **Commits**: Conventional Commits (`feat:`, `fix:`, `docs:`, `ci:`, `test:`, `chore:`)
- **File naming**: `provider/resource_<name>.go`, `provider/data_source_<name>.go`
- **Tests**: `TestAcc<Resource>` for acceptance, `Test<Unit>` for unit
- **Linting**: golangci-lint with `.golangci.yml` config
- **Documentation**: edit schemas and `examples/`, then run `task docs`; do not hand-edit generated schema sections
- **Dependencies**: prefer the standard library and existing modules; add a dependency only when its maintenance and security cost is justified
