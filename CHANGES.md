# Changes from upstream

This file documents all modifications made by Boozt Fashion AB to the original
ipam-autopilot project by Google LLC.

Original source: https://github.com/GoogleCloudPlatform/professional-services/tree/main/tools/ipam-autopilot

---

## Infrastructure / Build

- **Go module paths** renamed from `github.com/GoogleCloudPlatform/professional-services/*`
  to `github.com/boozt-platform/ipam-autopilot/{container,provider}`
- **Go version** updated from 1.14 to 1.26; all dependencies upgraded to latest
- **Added `.goreleaser.yaml`** — builds `terraform-provider-ipam-autopilot` (multi-platform)
  and `ipam-autopilot` (linux/amd64, linux/arm64); archives in OpenTofu registry format
- **Added `.github/workflows/release.yml`** — automated releases via go-semantic-release
  and conventional commits on merge to main

## API

- **Added `/api/v1` route group** — all IPAM endpoints now available under `/api/v1/ranges`
  and `/api/v1/domains`; legacy paths kept for Terraform provider backward compatibility
- **Added structured logging** — `log/slog` with JSON output (text via `LOG_FORMAT=text`),
  request ID and access log on every request
- **Added OpenTelemetry tracing** — OTLP gRPC exporter; noop when
  `OTEL_EXPORTER_OTLP_ENDPOINT` is unset
- **Updated Dockerfile** — Go 1.26, `distroless/static-debian12`

## Development / Testing

- **Added `docker-compose.yml`** — MySQL 8.4 + API + Jaeger for local development
- **Added integration tests** — testcontainers-go (MySQL); covers all domain and range
  endpoints including legacy route backward compat
- **Added `.devcontainer`** — Go 1.26, golangci-lint, docker-in-docker; consistent dev
  environment via VS Code Dev Containers
- **Added `Makefile`** — `make lint`, `make test`, `make test-integration`,
  `make dev-apply/destroy` for local Terraform testing against docker-compose
- **Added `.golangci.yml`** — golangci-lint v2 config; CI lint step on every PR

## Phase 3: Features

- **Added labels/metadata on IP ranges** — new `labels JSON` column on `subnets` table
  (migration `1773964800_add_labels_to_subnets`); API accepts and returns `labels` map on
  `POST /ranges` and `GET /ranges/:id`; Terraform provider exposes `labels` attribute on
  `ipam_ip_range` resource
- **Added `ipam_ip_range` data source** — look up an existing range by name without
  allocating a new one; uses `GET /ranges?name=` filter added to the API
- **Added `name` validation** — `POST /ranges` returns 400 if `name` is missing
- **Added `labels` validation** — `POST /ranges` returns 400 if any label key or value is empty
- **Expanded POST /ranges response** — now returns `id`, `name`, `cidr`, and `labels`

## Phase 4: Audit log

- **Added audit log** — new `audit_logs` table (migration `1773964900_create_audit_logs_table`);
  `GET /api/v1/audit?limit=N` returns last N events (default 100, max 1000); events written on
  every create/delete of ranges and routing domains, including `cidr` in detail for ranges
- **Refactored container into `server` sub-package** — all app logic moved to `container/server/`
  (`package server`); `container/main.go` is now a thin entry point; enables integration tests to
  import `server.NewApp` and `server.MigrateDatabase` without `package main` restrictions
- **Embedded SQL migrations** — `//go:embed migrations/*.sql` + `iofs` source; migrations are
  baked into the binary; `Dockerfile` no longer copies the `migrations/` directory separately
- **Refactored integration tests into `container/tests/`** — split into topic files
  (`domains_test.go`, `ranges_test.go`, `audit_log_test.go`, `legacy_test.go`, `helpers_test.go`);
  unit tests for subnet logic moved to `container/server/subnet_test.go`

## Phase 5: Bulk import + env variable consistency

- **Added `POST /api/v1/ranges/import`** — import pre-existing CIDRs into IPAM without
  auto-allocation; accepts an array of `{ name, cidr, domain?, parent?, labels? }` items;
  idempotent: ranges already present in the same domain are silently skipped; returns
  `{ imported, skipped, errors }` summary; each imported range is written to the audit log;
  use case: register manually-assigned subnets before enabling IPAM management, then
  `terraform import` them into Terraform state
- **Added `IPAM_DISABLE_BULK_IMPORT`** — set `TRUE` to disable the import endpoint (e.g. in
  production environments where arbitrary CIDR registration is undesirable)
- **Renamed all env variables to use `IPAM_` prefix** — `DATABASE_USER` →
  `IPAM_DATABASE_USER`, `CAI_ORG_ID` → `IPAM_CAI_ORG_ID`, `GCP_IDENTITY_TOKEN` →
  `IPAM_IDENTITY_TOKEN`, etc.; `OTEL_EXPORTER_OTLP_ENDPOINT` kept as-is (OpenTelemetry
  standard); full list in README
- **Added CAI integration documentation** — explains VPC scoping per routing domain and
  how CAI prevents collision with subnets not registered in IPAM
- **Added bulk import workflow documentation** — step-by-step guide for adopting an
  existing VPC into IPAM management including `terraform import` steps

## Phase 6: CAI integration refactor

- **Replaced live CAI API call with DB-backed cache** — new `cai_subnets` table
  (migration `1773939600_create_cai_subnets_table`); allocation reads from DB instead of
  calling Cloud Asset Inventory on every `POST /ranges`; eliminates ~3000 subnet round-trip
  on each auto-allocation
- **Added startup CAI sync + background loop** — on startup (when `IPAM_CAI_ORG_ID` is set)
  the server runs an initial blocking sync then starts a background goroutine that re-syncs
  on `IPAM_CAI_SYNC_INTERVAL` (default `5m`); no external scheduler needed
- **Sync is chunked and transactional** — upserts in batches of 200 per transaction; a
  failed chunk is retried on the next sync cycle; stale entries (subnets deleted in GCP) are
  removed after each sync
- **Fixed `log.Fatal` in CAI iterator** — a CAI API error during iteration now returns an
  error instead of crashing the server process
- **Fixed VPC name matching** — routing domains may now store the short VPC name (e.g.
  `my-vpc`) or the full resource URL; both forms match correctly against CAI network URLs
- **Added `IPAM_CAI_SYNC_INTERVAL`** — configurable sync interval (Go duration string,
  default `5m`)
- **Added CAI integration tests** — mock-free: tests seed `cai_subnets` directly and verify
  allocation avoids seeded CIDRs; covers short-name matching and VPC isolation

## Planned changes (not yet implemented)

See skill documentation for full roadmap:
- Provider registry migration to registry.opentofu.org
- Bulk import endpoint
