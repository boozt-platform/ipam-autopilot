---
page_title: "Getting Started"
description: |-
  Deploy the IPAM Autopilot backend and allocate your first IP range.
---

# Getting Started

This guide walks through deploying the IPAM Autopilot backend to GCP and allocating IP ranges with Terraform.

## 1. Deploy the backend

Use the `modules/ipam` module from the [boozt-platform/ipam-autopilot](https://github.com/boozt-platform/ipam-autopilot) repository:

```hcl
module "ipam" {
  source = "github.com/boozt-platform/ipam-autopilot//modules/ipam?ref=v1.8.0"

  project_id = "my-gcp-project"
  region     = "europe-west1"
}

output "ipam_url" {
  value = module.ipam.cloud_run_url
}
```

Run `tofu init && tofu apply` and note the `ipam_url` output.

## 2. Configure the provider

```hcl
terraform {
  required_providers {
    ipam = {
      source  = "boozt-platform/ipam-autopilot"
      version = "~> 1.8"
    }
  }
}

provider "ipam" {
  url = "https://your-ipam-cloud-run-url"
}
```

## 3. Create a routing domain and allocate ranges

```hcl
resource "ipam_routing_domain" "prod" {
  name = "prod"
  vpcs = ["prod-vpc"]
}

resource "ipam_ip_range" "root" {
  name       = "prod-root"
  range_size = 16
  domain     = ipam_routing_domain.prod.id
}

resource "ipam_ip_range" "gke_nodes" {
  name       = "prod-gke-nodes"
  range_size = 22
  domain     = ipam_routing_domain.prod.id
  parent     = ipam_ip_range.root.cidr
  labels     = { env = "prod", purpose = "gke-nodes" }
}

output "gke_nodes_cidr" {
  value = ipam_ip_range.gke_nodes.cidr
}
```

## 4. Read ranges from other stacks

Use the data source to read an allocated range without owning it:

```hcl
data "ipam_ip_range" "gke_nodes" {
  name = "prod-gke-nodes"
}
```

## Authentication

The provider uses [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) to obtain a Google identity token for the Cloud Run backend.

Run once on your workstation:

```bash
gcloud auth application-default login
```

For CI/CD, set `IPAM_IDENTITY_TOKEN` to a valid identity token scoped to the Cloud Run service URL.
