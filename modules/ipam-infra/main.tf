# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

locals {
  sa_email    = google_service_account.ipam.email
  db_instance = var.create_database ? module.mysql[0].instance_connection_name : var.database_instance_connection_name
}

# ── APIs ───────────────────────────────────────────────────────────────────────

resource "google_project_service" "apis" {
  for_each = toset(concat(
    [
      "iam.googleapis.com",
      "run.googleapis.com",
      "compute.googleapis.com",
      "cloudasset.googleapis.com",
      "sqladmin.googleapis.com",
      "servicenetworking.googleapis.com",
    ],
    can(regex("docker\\.pkg\\.dev", var.image)) ? ["artifactregistry.googleapis.com"] : []
  ))
  project = var.project_id
  service = each.key

  disable_on_destroy = false
}

data "google_compute_network" "vpc" {
  name    = var.network
  project = var.project_id
}

# ── Service Account ────────────────────────────────────────────────────────────

resource "google_service_account" "ipam" {
  project      = var.project_id
  account_id   = "ipam-autopilot"
  display_name = "IPAM Autopilot"

  depends_on = [google_project_service.apis]
}

# ── IAM ────────────────────────────────────────────────────────────────────────

resource "google_project_iam_member" "sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${local.sa_email}"
}

resource "google_project_iam_member" "sql_instance_user" {
  project = var.project_id
  role    = "roles/cloudsql.instanceUser"
  member  = "serviceAccount:${local.sa_email}"
}

resource "google_organization_iam_member" "cai_viewer" {
  count  = var.organization_id != "" ? 1 : 0
  org_id = var.organization_id
  role   = "roles/cloudasset.viewer"
  member = "serviceAccount:${local.sa_email}"
}

resource "google_cloud_run_v2_service_iam_member" "invoker" {
  count    = var.cloud_run_allow_unauthenticated ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.ipam.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# ── Private Service Access (VPC peering for Cloud SQL private IP) ──────────────

module "private_service_access" {
  count  = var.cloud_sql_private_ip ? 1 : 0
  source = "GoogleCloudPlatform/sql-db/google//modules/private_service_access"

  project_id    = var.project_id
  vpc_network   = data.google_compute_network.vpc.name
  prefix_length = 16

  depends_on = [google_project_service.apis]
}

# ── Cloud SQL (safer_mysql — IAM-only auth) ────────────────────────────────────

module "mysql" {
  count   = var.create_database ? 1 : 0
  source  = "GoogleCloudPlatform/sql-db/google//modules/safer_mysql"
  version = "~> 28.0"

  project_id         = var.project_id
  name               = var.database_instance_name
  database_version   = var.database_version
  region             = var.region
  zone               = var.zone
  tier               = var.database_tier
  db_name            = var.database_name
  db_collation       = var.db_collation
  vpc_network        = data.google_compute_network.vpc.id
  allocated_ip_range = var.cloud_sql_private_ip ? module.private_service_access[0].google_compute_global_address_name : null

  deletion_protection         = var.database_deletion_protection
  deletion_protection_enabled = var.database_deletion_protection
  user_labels                 = var.labels

  database_flags = [
    {
      name  = "cloudsql_iam_authentication"
      value = "on"
    },
    {
      name  = "wait_timeout"
      value = "300"
    },
    {
      name  = "interactive_timeout"
      value = "300"
    },
  ]

  iam_users = [
    {
      id    = trimsuffix(local.sa_email, ".gserviceaccount.com")
      email = local.sa_email
    }
  ]

  backup_configuration = var.database_backup_configuration

  depends_on = [module.private_service_access]
}

# ── Cloud Run v2 ───────────────────────────────────────────────────────────────

resource "google_cloud_run_v2_service" "ipam" {
  project             = var.project_id
  name                = var.cloud_run_name
  location            = var.region
  ingress             = var.cloud_run_ingress
  labels              = var.labels
  deletion_protection = var.cloud_run_deletion_protection

  template {
    service_account = local.sa_email
    labels          = var.labels

    scaling {
      max_instance_count = var.cloud_run_max_instances
    }

    dynamic "vpc_access" {
      for_each = var.cloud_sql_private_ip ? [1] : []
      content {
        egress = "PRIVATE_RANGES_ONLY"
        network_interfaces {
          network = data.google_compute_network.vpc.name
        }
      }
    }

    # Shared volume for Cloud SQL Auth Proxy Unix socket
    volumes {
      name = "cloudsql-socket"
      empty_dir {
        medium     = "MEMORY"
        size_limit = "32Mi"
      }
    }

    # Sidecar: Cloud SQL Auth Proxy v2 with IAM auto-auth
    containers {
      name  = "cloudsql-proxy"
      image = var.cloud_sql_proxy_image
      args = [
        "--auto-iam-authn",
        "--private-ip",
        "--unix-socket=/cloudsql",
        "--health-check",
        "--http-address=0.0.0.0",
        local.db_instance,
      ]
      volume_mounts {
        name       = "cloudsql-socket"
        mount_path = "/cloudsql"
      }
      startup_probe {
        period_seconds    = 1
        failure_threshold = 60
        http_get {
          path = "/startup"
          port = 9090
        }
      }
    }

    # Main container: IPAM Autopilot
    containers {
      name       = "ipam"
      image      = var.image
      depends_on = ["cloudsql-proxy"]

      env {
        name  = "IPAM_DATABASE_NET"
        value = "unix"
      }
      env {
        name  = "IPAM_DATABASE_HOST"
        value = "/cloudsql/${local.db_instance}"
      }
      env {
        name  = "IPAM_DATABASE_USER"
        value = split("@", local.sa_email)[0]
      }
      env {
        name  = "IPAM_DATABASE_NAME"
        value = var.database_name
      }
      env {
        name  = "IPAM_DISABLE_DATABASE_MIGRATION"
        value = var.disable_database_migration ? "TRUE" : "FALSE"
      }

      dynamic "env" {
        for_each = var.organization_id != "" ? [1] : []
        content {
          name  = "IPAM_CAI_ORG_ID"
          value = var.organization_id
        }
      }

      volume_mounts {
        name       = "cloudsql-socket"
        mount_path = "/cloudsql"
      }

      ports {
        container_port = 8080
        name           = "http1"
      }
    }
  }

  depends_on = [
    google_project_iam_member.sql_client,
    google_project_iam_member.sql_instance_user,
    module.mysql, # no-op when create_database = false
  ]
}
