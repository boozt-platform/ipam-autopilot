# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# AR remote repository proxying ghcr.io so Cloud Run v2 can pull the image
resource "google_artifact_registry_repository" "ghcr_proxy" {
  project       = var.project_id
  location      = var.region
  repository_id = "ghcr-proxy"
  description   = "Remote proxy for ghcr.io/boozt-platform/ipam-autopilot"
  format        = "DOCKER"
  mode          = "REMOTE_REPOSITORY"

  remote_repository_config {
    docker_repository {
      custom_repository {
        uri = "https://ghcr.io"
      }
    }
  }
}

module "ipam" {
  source = "../../modules/ipam-infra"

  project_id      = var.project_id
  region          = var.region
  organization_id = var.organization_id

  image = "${var.region}-docker.pkg.dev/${var.project_id}/ghcr-proxy/boozt-platform/ipam-autopilot:latest"

  database_deletion_protection    = false
  cloud_run_ingress               = "INGRESS_TRAFFIC_ALL"
  cloud_run_allow_unauthenticated = true
  database_tier                   = "db-custom-2-3840"
}
