# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# Example: reserve IP ranges in IPAM then create matching GCP VPC subnets.
#
# Deploy the IPAM service first:
#   cd ../infra && tofu apply
#
# Then apply this example:
#   tofu init
#   tofu apply \
#     -var="ipam_url=$(cd ../infra && tofu output -raw ipam_url)" \
#     -var="project_id=<your-project>"

# ── IPAM: reserve address space ───────────────────────────────────────────────

module "sandbox_network" {
  source = "../../modules/ipam-network"

  domain = {
    name = "sandbox-vpc"
    cidr = "10.0.0.0/8"
  }
  labels = { env = "sandbox" }

  networks = {
    "tenant"        = { size = 16 }
    "tenant-b"      = { size = 24 }
    "gke-nodes"     = { size = 16, labels = { team = "sre", env = "dev" } }
    "gke-pods"      = { size = 16 }
    "gke-services"  = { size = 16 }
    "mgmt"          = { size = 26 }
    "vpn-gw"        = { size = 27 }
    "proxy"         = { size = 28 }
    "nat"           = { size = 28 }
  }
}

# ── GCP: VPC ──────────────────────────────────────────────────────────────────

resource "google_compute_network" "sandbox" {
  project                 = var.project_id
  name                    = "sandbox-vpc"
  auto_create_subnetworks = false
}

# ── GCP: subnets — CIDRs come from IPAM ──────────────────────────────────────

resource "google_compute_subnetwork" "networks" {
  for_each = module.sandbox_network.networks

  project       = var.project_id
  region        = var.region
  network       = google_compute_network.sandbox.id
  name          = each.value.name
  ip_cidr_range = each.value.cidr
}
