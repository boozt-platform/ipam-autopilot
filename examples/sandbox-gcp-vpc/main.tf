# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# Sandbox example: creates a new GCP project and deploys the IPAM backend.
# After applying, use examples/sandbox-network to register address space.
#
#   tofu init
#   tofu apply \
#     -var="billing_account=<BILLING_ACCOUNT_ID>" \
#     -var="org_id=<ORG_ID>"

# ── Project ───────────────────────────────────────────────────────────────────

resource "random_id" "project_suffix" {
  byte_length = 4
}

resource "google_project" "sandbox" {
  name              = "ipam-sandbox"
  project_id        = "ipam-sandbox-${random_id.project_suffix.hex}"
  org_id            = var.org_id
  billing_account   = var.billing_account
  deletion_policy   = "DELETE"
}

# ── IPAM backend ──────────────────────────────────────────────────────────────

module "ipam" {
  source = "../../modules/ipam-infra"

  project_id = google_project.sandbox.project_id
  region     = var.region
  zone       = var.zone
  network    = google_compute_network.sandbox.name
  subnetwork = google_compute_subnetwork.ipam.name

  image                           = "docker.io/booztpl/ipam-autopilot:latest"
  cloud_run_allow_unauthenticated = true
  cloud_run_ingress               = "INGRESS_TRAFFIC_ALL"
  database_deletion_protection    = false
  database_tier                   = "db-g1-small"

  labels = { env = "sandbox" }

  module_depends_on = [google_project.sandbox.project_id]
}

# ── APIs (needed before VPC creation) ────────────────────────────────────────

resource "google_project_service" "compute" {
  project            = google_project.sandbox.project_id
  service            = "compute.googleapis.com"
  disable_on_destroy = false
}

# ── VPC ───────────────────────────────────────────────────────────────────────

resource "google_compute_network" "sandbox" {
  project                 = google_project.sandbox.project_id
  name                    = "sandbox-vpc"
  auto_create_subnetworks = false

  depends_on = [google_project_service.compute]
}

resource "google_compute_subnetwork" "ipam" {
  project       = google_project.sandbox.project_id
  region        = var.region
  network       = google_compute_network.sandbox.id
  name          = "ipam"
  ip_cidr_range = "10.255.0.0/24"
}
