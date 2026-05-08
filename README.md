# IPAM Autopilot

> **Fork notice:** This is a Boozt Fashion AB fork of
> [GoogleCloudPlatform/professional-services](https://github.com/GoogleCloudPlatform/professional-services/tree/main/tools/ipam-autopilot).
> See [NOTICE](./NOTICE) and [CHANGES.md](./CHANGES.md) for details.

IPAM Autopilot manages IP address space for GCP VPCs via a REST API and a Terraform provider. It auto-allocates non-overlapping CIDRs and integrates with Cloud Asset Inventory to avoid conflicts with subnets that exist outside of IPAM.

![Architecture showing Terraform, CloudRun, CloudSQL and Cloud Asset Inventory](./img/architecture.png "IPAM Autopilot Architecture")

---

## Getting started

### 1. Deploy the backend

Deploy the IPAM backend (Cloud Run + Cloud SQL) into your GCP project:

```hcl
module "ipam" {
  source = "github.com/boozt-platform/ipam-autopilot//modules/ipam-infra?ref=v1.15.0"

  project_id = "my-project"
  region     = "europe-west1"
  network    = "my-vpc"
  subnetwork = "my-ipam-subnet"
}

output "ipam_url" {
  value = module.ipam.service_url
}
```

For a full working example including project and VPC creation see [`examples/sandbox-gcp-vpc`](./examples/sandbox-gcp-vpc).

### 2. Register a network

Register a VPC and allocate CIDR blocks using the Terraform provider:

```hcl
terraform {
  required_providers {
    ipam = {
      source  = "boozt-platform/ipam-autopilot"
      version = "~> 1.15"
    }
  }
}

provider "ipam" {
  url = "https://your-ipam-url"  # or set IPAM_URL env var
}

module "prod_network" {
  source = "github.com/boozt-platform/ipam-autopilot//modules/ipam-network?ref=v1.15.0"

  domain = {
    name = "prod-vpc"
    cidr = "10.0.0.0/8"
  }

  networks = {
    "gke-nodes"    = { size = 16 }
    "gke-services" = { size = 24 }
    "tenant"       = { size = 16 }
  }
}
```

For a full working example see [`examples/sandbox-network`](./examples/sandbox-network).

---

## Documentation

- [Getting started guide](./docs/guides/getting-started.md): authentication, provider setup, first allocation
- [`modules/ipam-infra`](./modules/ipam-infra): all backend module variables and outputs
- [`modules/ipam-network`](./modules/ipam-network): all network module variables and outputs
- [Terraform provider registry](https://registry.terraform.io/providers/boozt-platform/ipam-autopilot/latest/docs): provider resources and data sources

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for dev environment setup, testing, and commit conventions.
