# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# Registers a VPC domain and network blocks in IPAM, then creates matching
# GCP subnets. Run after examples/sandbox-gcp-vpc:
#
#   PROJECT_ID=$(tofu -chdir=../sandbox-gcp-vpc output -raw project_id)
#   IPAM_URL=$(tofu -chdir=../sandbox-gcp-vpc output -raw ipam_url)
#
#   tofu init
#   tofu apply \
#     -var="project_id=$PROJECT_ID" \
#     -var="ipam_url=$IPAM_URL"

# ── IPAM: register address space ──────────────────────────────────────────────

module "network" {
  source = "../../modules/ipam-network"

  domain = {
    name = var.network
    cidr = "10.0.0.0/8"
  }
  labels = { env = "sandbox" }

  networks = {
    "tenant"       = { size = 16 }
    "tenant-b"     = { size = 24 }
    "gke-nodes"    = { size = 16, labels = { team = "sre", env = "dev" } }
    "gke-pods"     = { size = 16 }
    "gke-services" = { size = 16 }
    "mgmt"         = { size = 26 }
    "vpn-gw"       = { size = 27 }
    "proxy"        = { size = 28 }
    "nat"          = { size = 28 }
  }
}

# ── GCP: subnets — CIDRs come from IPAM ──────────────────────────────────────

data "google_compute_network" "vpc" {
  name    = var.network
  project = var.project_id
}

resource "google_compute_subnetwork" "networks" {
  for_each = module.network.networks

  project       = var.project_id
  region        = var.region
  network       = data.google_compute_network.vpc.id
  name          = each.value.name
  ip_cidr_range = each.value.cidr
}
