# Changes from upstream

This file documents all modifications made by Boozt Fashion AB to the original
ipam-autopilot project by Google LLC.

Original source: https://github.com/GoogleCloudPlatform/professional-services/tree/main/tools/ipam-autopilot

---

## Docker Publishing

- Added `container/Dockerfile` - self-contained build (no infrastructure/output dependency)
- Added `.github/workflows/docker-publish.yml` - publishes multi-arch image (`linux/amd64`, `linux/arm64`) to `ghcr.io/boozt-platform/ipam-autopilot` on each GitHub release
- Added OCI metadata labels to container image: `title`, `description`, `licenses`, `vendor` (plus `version`, `revision`, `created`, `source` auto-set by `docker/metadata-action`)
- Added Docker Hub publishing to `booztpl/ipam-autopilot` alongside GHCR on each release

---

## Infrastructure / Build

- Go module paths renamed from `github.com/GoogleCloudPlatform/professional-services/*` to `github.com/boozt-platform/ipam-autopilot/{container,provider}`
- Go version updated from 1.14 to 1.26; all dependencies upgraded to latest
- Added `.goreleaser.yaml` - builds `terraform-provider-ipam-autopilot` (multi-platform) and `ipam-autopilot` (linux/amd64, linux/arm64); archives in OpenTofu registry format
- Added `.github/workflows/release.yml` - automated releases via go-semantic-release and conventional commits on merge to main

## API

- Added `/api/v1` route group - all IPAM endpoints now available under `/api/v1/ranges` and `/api/v1/domains`; legacy paths kept for Terraform provider backward compatibility
- Added structured logging - `log/slog` with JSON output (text via `LOG_FORMAT=text`), request ID and access log on every request
- Added OpenTelemetry tracing - OTLP gRPC exporter; noop when `OTEL_EXPORTER_OTLP_ENDPOINT` is unset
- Updated Dockerfile - Go 1.26, `distroless/static-debian12`

## Development / Testing

- Added `docker-compose.yml` - MySQL 8.4 + API + Jaeger for local development
- Added integration tests - testcontainers-go (MySQL); covers all domain and range endpoints including legacy route backward compat
- Added `.devcontainer` - Go 1.26, golangci-lint, docker-in-docker; consistent dev environment via VS Code Dev Containers
- Added `Makefile` - `make lint`, `make test`, `make test-integration`, `make dev-apply/destroy` for local Terraform testing against docker-compose
- Added `.golangci.yml` - golangci-lint v2 config; CI lint step on every PR

## Phase 3: Features

- Added labels/metadata on IP ranges - new `labels JSON` column on `subnets` table (migration `1773964800_add_labels_to_subnets`); API accepts and returns `labels` map on `POST /ranges` and `GET /ranges/:id`; Terraform provider exposes `labels` attribute on `ipam_ip_range` resource
- Added `ipam_ip_range` data source - look up an existing range by name without allocating a new one; uses `GET /ranges?name=` filter added to the API
- Added `name` validation - `POST /ranges` returns 400 if `name` is missing
- Added `labels` validation - `POST /ranges` returns 400 if any label key or value is empty
- Expanded `POST /ranges` response - now returns `id`, `name`, `cidr`, and `labels`

## Phase 4: Audit log

- Added audit log - new `audit_logs` table (migration `1773964900_create_audit_logs_table`); `GET /api/v1/audit?limit=N` returns last N events (default 100, max 1000); events written on every create/delete of ranges and routing domains, including `cidr` in detail for ranges
- Refactored container into `server` sub-package - all app logic moved to `container/server/` (`package server`); `container/main.go` is now a thin entry point; enables integration tests to import `server.NewApp` and `server.MigrateDatabase` without `package main` restrictions
- Embedded SQL migrations - `//go:embed migrations/*.sql` + `iofs` source; migrations are baked into the binary; `Dockerfile` no longer copies the `migrations/` directory separately
- Refactored integration tests into `container/tests/` - split into topic files (`domains_test.go`, `ranges_test.go`, `audit_log_test.go`, `legacy_test.go`, `helpers_test.go`); unit tests for subnet logic moved to `container/server/subnet_test.go`

## Phase 5: Bulk import + env variable consistency

- Added `POST /api/v1/ranges/import` - import pre-existing CIDRs into IPAM without auto-allocation; accepts an array of `{ name, cidr, domain?, parent?, labels? }` items; idempotent: ranges already present in the same domain are silently skipped; returns `{ imported, skipped, errors }` summary; each imported range is written to the audit log
- Added `IPAM_DISABLE_BULK_IMPORT` - set `TRUE` to disable the import endpoint in production environments where arbitrary CIDR registration is undesirable
- Renamed all env variables to use `IPAM_` prefix - `DATABASE_USER` -> `IPAM_DATABASE_USER`, `CAI_ORG_ID` -> `IPAM_CAI_ORG_ID`, `GCP_IDENTITY_TOKEN` -> `IPAM_IDENTITY_TOKEN`, etc.; `OTEL_EXPORTER_OTLP_ENDPOINT` kept as-is (OpenTelemetry standard); full list in README
- Added CAI integration documentation - explains VPC scoping per routing domain and how CAI prevents collision with subnets not registered in IPAM
- Added bulk import workflow documentation - step-by-step guide for adopting an existing VPC into IPAM management including `terraform import` steps

## Phase 6: CAI integration refactor

- Replaced live CAI API call with DB-backed cache - new `cai_subnets` table (migration `1773939600_create_cai_subnets_table`); allocation reads from DB instead of calling Cloud Asset Inventory on every `POST /ranges`; eliminates ~3000 subnet round-trip on each auto-allocation
- Added startup CAI sync + background loop - on startup (when `IPAM_CAI_ORG_ID` is set) the server runs an initial blocking sync then starts a background goroutine that re-syncs on `IPAM_CAI_SYNC_INTERVAL` (default `5m`); no external scheduler needed
- Sync is chunked and transactional - upserts in batches of 200 per transaction; a failed chunk is retried on the next sync cycle; stale entries (subnets deleted in GCP) are removed after each sync
- Fixed `log.Fatal` in CAI iterator - a CAI API error during iteration now returns an error instead of crashing the server process
- Fixed VPC name matching - routing domains may now store the short VPC name (e.g. `my-vpc`) or the full resource URL; both forms match correctly against CAI network URLs
- Added `IPAM_CAI_SYNC_INTERVAL` - configurable sync interval (Go duration string, default `5m`)
- Added CAI integration tests - mock-free: tests seed `cai_subnets` directly and verify allocation avoids seeded CIDRs; covers short-name matching and VPC isolation

## Infrastructure / Terraform module

- Added `modules/ipam-infra` Terraform module - full GCP deployment: Cloud SQL (safer_mysql), Cloud Run v2 with Cloud SQL Auth Proxy sidecar, Service Account with IAM roles, optional Cloud Asset Inventory org-level viewer role; no hardcoded credentials, IAM authentication via `--auto-iam-authn` proxy flag
- Added Cloud SQL Auth Proxy sidecar - runs as a sidecar container sharing a Unix socket volume; handles IAM token exchange transparently; `IPAM_DATABASE_PASSWORD` is intentionally unset (proxy provides the token); image version exposed via `cloud_sql_proxy_image` variable (default `2.21.2`)
- Enabled Cloud SQL IAM authentication - `cloudsql_iam_authentication=on` database flag; Service Account email used as MySQL username (prefix before `@`)
- Added `cloud_run_ingress` variable - configurable ingress setting (default: `INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER`; infra example uses `INGRESS_TRAFFIC_ALL`)
- Added `cloud_sql_proxy_image` variable - pinnable Cloud SQL Auth Proxy image version (default `gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.21.2`)
- Added MySQL `wait_timeout`/`interactive_timeout` flags - set to 300s to terminate idle connections and prevent zombie sessions from holding row locks
- Fixed `FOR UPDATE` lock on routing_domains - `GetDefaultRoutingDomainFromDB` no longer takes a row lock on the routing_domains table; previously serialized all concurrent allocation requests and caused `Lock wait timeout exceeded (1205)` under load
- Added `db.SetConnMaxLifetime(4 * time.Minute)` - Go connection pool recycles connections before MySQL's 300s wait_timeout expires, preventing stale connection errors
- Added `AllowCleartextPasswords: true` - required for MySQL IAM token auth flow via Cloud SQL Auth Proxy; token is transmitted in cleartext over the local Unix socket
- Added `examples/infra` - sandbox deployment example for testing (`INGRESS_TRAFFIC_ALL`, `database_deletion_protection = false`, AR remote proxy for ghcr.io)

## Provider fixes

- Fixed `authorized_user` identity token fallback - `getIdentityToken()` error check was comparing against a stale error string; updated to `strings.Contains(..., "authorized_user")` so developer ADC credentials (laptop / `gcloud auth application-default login`) work correctly
- Fixed Cloud Run identity token audience - provider previously used a hardcoded audience `http://ipam-autopilot.com`; now passes the actual Cloud Run service URL as audience so tokens are accepted by authenticated (non-public) Cloud Run services
- Added `cloud_run_allow_unauthenticated` variable - optional `allUsers` `roles/run.invoker` binding on the Cloud Run service; default `false`; enable for sandbox/testing only
- Added `database_version` validation to `modules/ipam-infra` - must match `^MYSQL_`
- Updated README - removed outdated `infrastructure/` deployment instructions and GCS registry setup; documented `modules/ipam-infra` and `modules/ipam-network` usage, authentication, and all environment variables

## Terraform modules redesign

- Renamed `modules/ipam` to `modules/ipam-infra` for clarity
- Replaced `modules/ipam-client` with new `modules/ipam-network` - domain-centric design: one routing domain with a root CIDR block, network blocks carved from it
- Added `database_backup_configuration` variable to `modules/ipam-infra` - exposes all backup settings as a configurable object with defaults; added validation that `retained_backups > transaction_log_retention_days` when backup is enabled
- Added `domain` object variable to `modules/ipam-network` - combines domain name and root CIDR into a single input (`name` + `cidr`)
- Added per-network label override to `modules/ipam-network` - module-level labels are inherited by all network blocks; per-network labels fully override them when set
- Added `README.md` per module generated via `terraform-docs`
- Added `.terraform-docs.yml` at repo root - single config for all modules
- Added `make docs-modules` target - regenerates README for all modules
- Added `make update-version VERSION=x.y.z` target - updates all `?ref=` and provider version constraints across the repo
- Renamed `examples/sandbox` to `examples/infra`
- Renamed `examples/sandbox-client` to `examples/sandbox-gcp-vpc` - extended with GCP VPC and subnet creation using IPAM-allocated CIDRs
- Removed outdated examples: `simple-example`, `vpc-example`, `example-with-multiple-ranges`

## Provider fixes (cont.)

- Derived `range_size` from `cidr` prefix when `cidr` is set - `range_size` is now optional when `cidr` is provided; the prefix length is parsed automatically; `range_size` remains required for auto-allocation (when `cidr` is omitted)
- Removed `range_size` workaround from `modules/ipam-network` - no longer needed now that the provider derives it

## Dockerfile hardening + lint enforcement

- Switched to `gcr.io/distroless/static-debian12:nonroot` base image - runs as `nonroot` (UID 65532) by default; explicit `USER nonroot:nonroot` instruction added for clarity
- Added `hadolint` Dockerfile linting - `make lint-docker` runs locally; `hadolint/hadolint-action@v3.1.0` added to CI lint job
- Added `.pre-commit-config.yaml` - `hadolint` and `gofmt` run automatically on `git commit` after `pre-commit install`
- Updated `.devcontainer/install-tools.sh` - added `hadolint`, `opentofu`, `terraform-docs`, `pre-commit`; updated help text
- Added `exiasr.hadolint` VS Code extension to devcontainer

## In-place label updates

- Added `PUT /api/v1/ranges/:id` - updates labels on an existing range without destroying it; accepts `{ "labels": {...} }`, validates keys/values, writes an audit log entry
- Removed `ForceNew` from `labels` in `ipam_ip_range` provider resource - label changes now trigger an in-place update instead of destroy+recreate
- Added `resourceUpdate` to Terraform provider - calls `PUT /ranges/:id` with the new labels map

## Provider documentation

- Added `provider/docs/` - OpenTofu registry-compatible documentation generated via `terraform-plugin-docs`; covers provider index, `ipam_ip_range` resource, `ipam_routing_domain` resource, `ipam_ip_range` data source, and a getting-started guide
- Added schema descriptions to all provider resource and data source fields so `tfplugindocs` generates accurate per-attribute documentation
- Added `provider/templates/` - custom doc templates with usage examples, import instructions, and cross-references
- Added `provider/examples/` - HCL example files referenced by doc templates
- Added `make docs` to provider Makefile - regenerates `docs/` from templates and schema
- Added `provider/tools/tools.go` - pins `tfplugindocs` as a Go tool dependency
