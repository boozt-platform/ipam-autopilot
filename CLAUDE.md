# IPAM Autopilot - Agent Instructions

This file is read automatically by Claude Code at the start of every session.

## Repository layout

| Path | What it is |
|---|---|
| `container/` | Go backend (Fiber API, MySQL, Cloud Asset Inventory) |
| `provider/` | Terraform/OpenTofu provider (terraform-plugin-sdk v2) |
| `modules/ipam-infra/` | Terraform module - deploys backend to GCP (Cloud Run + Cloud SQL) |
| `modules/ipam-network/` | Terraform module - registers a VPC domain + network blocks |
| `examples/` | Working usage examples |
| `docs/` | Generated provider docs (OpenTofu registry format) |
| `provider/templates/` | Source templates for docs (edit these, not `docs/`) |

## Full checklist for every feat or fix

Work through these steps in order. Do not commit until all pass.

### 1. Implementation

- Follow existing code patterns in the same file/package
- Backend changes: `container/server/api.go` (handler) + `container/server/data_access.go` (DB) + `container/server/server.go` (route registration)
- Provider changes: `provider/ipam/resources/resource_ip_range.go` or `resource_routing_domain.go`
- Module changes: `modules/ipam-network/main.tf` or `modules/ipam-infra/main.tf`

### 2. Tests

Write tests before committing. All three layers apply depending on what changed:

| Layer | Location | Command | When required |
|---|---|---|---|
| Go unit | `container/server/*_test.go`, `provider/ipam/resources/*_test.go` | `go test ./...` | Any Go logic change |
| Go integration | `container/tests/*_test.go` (build tag `integration`) | `make test-integration` | Any API endpoint change |
| HCL unit | `modules/*/tests/unit_test.tftest.hcl` | `make test` | Any module change |

### 3. Documentation

Update every location that describes the changed behaviour:

- `CHANGES.md` - always, one bullet per logical change, no bold, no em dashes
- `README.md` - if API, env vars, or module interface changed
- `provider/templates/resources/ip_range.md.tmpl` - if provider resource schema or behaviour changed
- `provider/templates/guides/getting-started.md.tmpl` - if usage examples are affected
- `docs/resources/ip_range.md` and `docs/guides/getting-started.md` - mirror changes from templates (or run `make docs`)
- `modules/*/README.md` - if module variables/outputs changed (run `make docs-modules`)

### 4. Run the full gate

```bash
make check
```

This runs: `lint` (golangci-lint + hadolint) + `fmt` (gofmt) + `test` (unit + HCL) + `build-provider`.

Fix every error before proceeding. Do not skip or suppress linter warnings without a documented reason.

### 5. Review checklist

Go:
- No `log.Fatal` outside `main()` - return errors instead
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`
- No global mutable state outside `var db *sql.DB` pattern
- New DB queries use `?` placeholders, never string interpolation

Terraform/OpenTofu:
- Variables have `description` and `type`
- Sensitive outputs marked `sensitive = true`
- New resources include a `validation` block where input can be invalid
- `ForceNew = true` only when the API truly cannot update in-place

Docker:
- Always use a pinned, non-root base image tag
- No secrets or credentials in image layers

### 6. Commit

Use conventional commits - go-semantic-release derives the version from them:

| Prefix | Version bump | When to use |
|---|---|---|
| `feat:` | minor | new capability visible to users |
| `fix:` | patch | bug fix |
| `docs:` | none | documentation only |
| `test:` | none | tests only |
| `chore:` | none | tooling, CI, dependencies |

Commit related files in logical groups (not all at once). Close issues with `Closes #N` in the commit body.

### 7. After merge

go-semantic-release automatically tags and releases on merge to `main`. After the new tag appears:

```bash
make update-version VERSION=vX.Y.Z
make docs
make docs-modules
```

Commit the version bump as `chore: update version references to vX.Y.Z`.

## Key design decisions

- `range_size` is derived from `cidr` prefix when `cidr` is set - do not add it redundantly in examples
- Labels on `ipam_ip_range` update in-place (no destroy+recreate) - document this in any usage example
- `name` on `ipam_ip_range` is `ForceNew` - it is the lookup key for `data "ipam_ip_range"` and audit log
- `modules/ipam-network` root range uses direct CIDR insert (no `parent`) - this is intentional
- Docker images published to both `ghcr.io/boozt-platform/ipam-autopilot` and `booztpl/ipam-autopilot`
