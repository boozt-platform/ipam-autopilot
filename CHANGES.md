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

## Planned changes (not yet implemented)

See skill documentation for full roadmap:
- Provider registry migration to registry.opentofu.org
- Bulk import endpoint
