# Contributing

## Repo layout

| Path | What it is |
|---|---|
| `container/` | Go backend: Fiber HTTP server, MySQL, Cloud Asset Inventory |
| `provider/` | Terraform/OpenTofu provider (terraform-plugin-sdk v2) |
| `modules/ipam-infra/` | Terraform module: deploys backend to GCP (Cloud Run + Cloud SQL) |
| `modules/ipam-network/` | Terraform module: registers a VPC domain and network blocks |
| `examples/` | Working usage examples |
| `docs/` | Generated provider docs (OpenTofu registry format). Do not edit directly. |
| `provider/templates/` | Source templates for `docs/`. Edit these, not `docs/`. |

## Dev environment

The easiest way is to open the repo in VS Code. The [devcontainer](./.devcontainer) includes Go, OpenTofu, golangci-lint, hadolint, terraform-docs, and gcloud.

For manual setup you need: Go 1.22+, OpenTofu or Terraform, Docker, golangci-lint, hadolint.

## Running the stack locally

```bash
docker compose up --build -d
```

Starts MySQL + the IPAM API at `localhost:8080` and Jaeger at `localhost:16686`.

To test the Terraform provider against the local stack:

```bash
make dev-apply    # builds provider, applies examples/local-dev
make dev-destroy  # tears it down
```

## Making changes

### Backend (Go)

| What changed | Files to edit |
|---|---|
| New API endpoint | `container/server/api.go` (handler) + `container/server/data_access.go` (DB) + `container/server/server.go` (route) |
| DB schema change | Add a migration file in `container/server/migrations/` |
| Provider resource | `provider/ipam/resources/resource_ip_range.go` or `resource_routing_domain.go` |

### Terraform modules

| Module | Main file |
|---|---|
| `ipam-infra` | `modules/ipam-infra/main.tf` |
| `ipam-network` | `modules/ipam-network/main.tf` |

## Testing

```bash
make test                # Go unit tests (container + provider) + HCL module tests
make test-integration    # integration tests via testcontainers (requires Docker)
make test-modules        # HCL unit tests only, using locally built provider binary
```

HCL module tests live in `modules/*/tests/unit_test.tftest.hcl`. They always run against the locally built provider binary, not the published registry version. Do not run `tofu test` directly in a module directory without the dev override.

Write or update tests for every change before running the gate.

## Pre-commit gate

```bash
make check
```

Runs lint (golangci-lint + hadolint) + format (gofmt) + tests + build + docs. Fix every failure before committing. Do not suppress linter warnings without a documented reason.

## Commit conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org/). go-semantic-release derives the version from commit prefixes automatically on merge to `main`.

| Prefix | Version bump | When to use |
|---|---|---|
| `feat:` | minor | new capability visible to users |
| `fix:` | patch | bug fix |
| `docs:` | none | documentation only |
| `test:` | none | tests only |
| `chore:` | none | tooling, CI, dependencies |

Commit related files in logical groups. Reference issues with `Closes #N` in the commit body.

## After a release

When go-semantic-release creates a new tag on `main`:

```bash
git checkout main && git pull
make update-version VERSION=vX.Y.Z
make docs
make docs-modules
git commit -am "chore: update version references to vX.Y.Z"
```
