# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

locals {
  sa_email    = "ipam-autopilot@${var.project_id}.iam.gserviceaccount.com"
  sa_username = split("@", local.sa_email)[0]
  db_instance = var.create_database ? module.mysql[0].instance_connection_name : var.database_instance_connection_name
  # Computed statically to avoid data source timing issues when the VPC is
  # created in the same apply as this module.
  network_id        = "projects/${var.project_id}/global/networks/${var.network}"
  network_self_link = "https://www.googleapis.com/compute/v1/projects/${var.project_id}/global/networks/${var.network}"
  subnetwork_name   = coalesce(var.subnetwork, var.network)
  # GRANT statement — changing this triggers the db_grant job to re-run.
  db_grant_sql = "GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, DROP, INDEX, LOCK TABLES, REFERENCES ON ${var.database_name}.* TO '${local.sa_username}'@'%';"
  # Module-managed databases are always private (safer_mysql enforces no public IP).
  # For existing databases (create_database = false), cloud_run_direct_vpc decides.
  use_direct_vpc = var.create_database ? true : var.cloud_run_direct_vpc
}

resource "terraform_data" "module_depends_on" {
  input = var.module_depends_on
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
    can(regex("docker\\.pkg\\.dev", var.image)) ? ["artifactregistry.googleapis.com"] : [],
    var.create_database ? ["secretmanager.googleapis.com"] : [],
  ))
  project = var.project_id
  service = each.key

  disable_on_destroy = false

  depends_on = [terraform_data.module_depends_on]
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

  depends_on = [google_service_account.ipam]
}

resource "google_project_iam_member" "sql_instance_user" {
  project = var.project_id
  role    = "roles/cloudsql.instanceUser"
  member  = "serviceAccount:${local.sa_email}"

  depends_on = [google_service_account.ipam]
}

resource "google_organization_iam_member" "cai_viewer" {
  count  = var.organization_id != "" ? 1 : 0
  org_id = var.organization_id
  role   = "roles/cloudasset.viewer"
  member = "serviceAccount:${local.sa_email}"

  depends_on = [google_service_account.ipam]
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
#
# Inlined instead of using the GoogleCloudPlatform/sql-db private_service_access
# module to avoid its data "google_compute_network" lookup, which fails during
# plan in a fresh project before the network is created. local.network_self_link
# is computed statically so no data source is needed.

resource "google_compute_global_address" "private_service_access" {
  count         = var.create_database ? 1 : 0
  project       = var.project_id
  name          = "google-managed-services-${var.network}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = local.network_self_link

  depends_on = [google_project_service.apis["servicenetworking.googleapis.com"]]
}

resource "google_service_networking_connection" "private_service_access" {
  count                   = var.create_database ? 1 : 0
  network                 = local.network_self_link
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_service_access[0].name]
  deletion_policy         = "ABANDON"

  depends_on = [google_project_service.apis["servicenetworking.googleapis.com"]]
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
  edition            = var.database_edition
  db_name            = var.database_name
  db_collation       = var.db_collation
  vpc_network        = local.network_id
  allocated_ip_range = google_compute_global_address.private_service_access[0].name

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

  iam_users = []

  backup_configuration = var.database_backup_configuration

  depends_on = [google_service_networking_connection.private_service_access]
}

# Create the IAM SQL user outside safer_mysql so its key is not in a for_each,
# which would fail at plan time when project_id is not yet known (e.g. when the
# GCP project itself is created in the same apply).
resource "google_sql_user" "iam_sa" {
  count    = var.create_database ? 1 : 0
  project  = var.project_id
  instance = module.mysql[0].instance_name
  name     = local.sa_email
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"

  depends_on = [module.mysql]
}

# ── Database GRANT setup ───────────────────────────────────────────────────────
#
# A dedicated service account (ipam-db-setup) stores the safer_mysql default
# user password in Secret Manager and runs a one-off Cloud Run Job that applies
# the minimum required MySQL privileges to the IPAM service account.
#
# The job re-runs automatically when the privilege set changes: db_grant_sql is
# hashed into triggers_replace, so updating the GRANT in a new module version
# forces a new terraform_data resource → provisioner re-executes → job runs.
#
# Requires gcloud to be installed and authenticated in the Terraform execution
# environment (local or CI/CD runner).

resource "google_service_account" "db_setup" {
  count        = var.create_database ? 1 : 0
  project      = var.project_id
  account_id   = "ipam-db-setup"
  display_name = "IPAM Database Setup"

  depends_on = [google_project_service.apis]
}

resource "google_project_iam_member" "db_setup_sql_client" {
  count   = var.create_database ? 1 : 0
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.db_setup[0].email}"

  depends_on = [google_service_account.db_setup]
}

resource "google_secret_manager_secret" "db_default_password" {
  count     = var.create_database ? 1 : 0
  project   = var.project_id
  secret_id = "${var.database_instance_name}-setup-password"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "db_default_password" {
  count       = var.create_database ? 1 : 0
  secret      = google_secret_manager_secret.db_default_password[0].id
  secret_data = module.mysql[0].generated_user_password

  depends_on = [module.mysql]
}

resource "google_secret_manager_secret_iam_member" "db_default_password_reader" {
  count     = var.create_database ? 1 : 0
  project   = var.project_id
  secret_id = google_secret_manager_secret.db_default_password[0].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.db_setup[0].email}"

  depends_on = [google_service_account.db_setup]
}

resource "google_cloud_run_v2_job" "db_grant" {
  count    = var.create_database ? 1 : 0
  project  = var.project_id
  name     = "${var.cloud_run_name}-db-grant"
  location = var.region

  deletion_protection = false

  template {
    template {
      service_account = google_service_account.db_setup[0].email
      max_retries     = 3

      dynamic "vpc_access" {
        for_each = [1]
        content {
          egress = "PRIVATE_RANGES_ONLY"
          network_interfaces {
            network    = var.network
            subnetwork = local.subnetwork_name
          }
        }
      }

      volumes {
        name = "cloudsql-socket"
        empty_dir {
          medium     = "MEMORY"
          size_limit = "32Mi"
        }
      }

      containers {
        name  = "cloudsql-proxy"
        image = var.cloud_sql_proxy_image
        args = [
          "--private-ip",
          "--unix-socket=/cloudsql",
          "--exit-zero-on-sigterm",
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

      containers {
        name       = "mysql-grant"
        image      = "docker.io/library/mysql:8.4"
        depends_on = ["cloudsql-proxy"]
        command    = ["/bin/sh", "-c"]
        args = [
          "mysql --socket=/cloudsql/${local.db_instance} -u default --password=\"$DB_PASSWORD\" -e \"${local.db_grant_sql}\""
        ]
        env {
          name = "DB_PASSWORD"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.db_default_password[0].secret_id
              version = "latest"
            }
          }
        }
        volume_mounts {
          name       = "cloudsql-socket"
          mount_path = "/cloudsql"
        }
      }
    }
  }

  depends_on = [
    google_project_iam_member.db_setup_sql_client,
    google_secret_manager_secret_iam_member.db_default_password_reader,
    google_secret_manager_secret_version.db_default_password,
  ]
}

data "google_client_config" "default" {}

resource "terraform_data" "run_db_grant" {
  count = var.create_database ? 1 : 0

  # Re-runs the job when the privilege set changes.
  triggers_replace = sha256(local.db_grant_sql)

  provisioner "local-exec" {
    environment = {
      TOKEN   = data.google_client_config.default.access_token
      PROJECT = var.project_id
      REGION  = var.region
      JOB     = google_cloud_run_v2_job.db_grant[0].name
    }
    command = <<-EOT
      set -e
      RESPONSE=$(curl -sf -X POST \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        "https://run.googleapis.com/v2/projects/$PROJECT/locations/$REGION/jobs/$JOB:run")

      OP=$(echo "$RESPONSE" | tr -d ' \n' | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)
      if [ -z "$OP" ]; then
        echo "Failed to parse operation name from response." >&2
        exit 1
      fi
      echo "Waiting for operation: $OP"

      for i in $(seq 1 30); do
        STATUS=$(curl -sf \
          -H "Authorization: Bearer $TOKEN" \
          "https://run.googleapis.com/v2/$OP")
        COMPACT=$(echo "$STATUS" | tr -d ' \n')
        if echo "$COMPACT" | grep -q '"done":true'; then
          if echo "$COMPACT" | grep -q '"error"'; then
            echo "GRANT job failed." >&2
            exit 1
          fi
          echo "GRANT job completed."
          exit 0
        fi
        echo "Attempt $i/30, retrying in 10s..."
        sleep 10
      done

      echo "Timeout waiting for GRANT job." >&2
      exit 1
    EOT
  }

  depends_on = [
    google_cloud_run_v2_job.db_grant,
    google_sql_user.iam_sa,
  ]
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
      for_each = local.use_direct_vpc ? [1] : []
      content {
        egress = "PRIVATE_RANGES_ONLY"
        network_interfaces {
          network    = var.network
          subnetwork = local.subnetwork_name
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
      args = concat(
        local.use_direct_vpc ? ["--private-ip"] : [],
        [
          "--auto-iam-authn",
          "--unix-socket=/cloudsql",
          "--health-check",
          "--http-address=0.0.0.0",
          local.db_instance,
        ]
      )
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
        value = local.sa_username
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
    module.mysql,                # no-op when create_database = false
    terraform_data.run_db_grant, # no-op when create_database = false
  ]
}
