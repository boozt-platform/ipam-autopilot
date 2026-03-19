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

## Planned changes (not yet implemented)

See skill documentation for full roadmap:
- Metadata/labels on IP ranges
- `ipam_ip_range` data source for Terraform provider
- Audit log endpoint
- Provider registry migration to registry.opentofu.org
- Bulk import endpoint
