# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

mock_provider "google" {
  mock_data "google_compute_network" {
    defaults = {
      id = "projects/test-project/global/networks/default"
    }
  }
}

# google-beta is used internally by some GoogleCloudPlatform modules
mock_provider "google-beta" {}

variables {
  project_id = "test-project"
}

# ── APIs ──────────────────────────────────────────────────────────────────────

run "enables_required_apis" {
  command = plan

  assert {
    condition = toset(keys(google_project_service.apis)) == toset([
      "iam.googleapis.com",
      "run.googleapis.com",
      "compute.googleapis.com",
      "cloudasset.googleapis.com",
      "sqladmin.googleapis.com",
      "servicenetworking.googleapis.com",
    ])
    error_message = "Should enable exactly the 6 required GCP APIs."
  }
}

run "enables_artifactregistry_api_when_image_is_from_ar" {
  command = plan

  variables {
    image = "europe-west1-docker.pkg.dev/test-project/ghcr-proxy/boozt-platform/ipam-autopilot:latest"
  }

  assert {
    condition     = contains(toset(keys(google_project_service.apis)), "artifactregistry.googleapis.com")
    error_message = "Should enable artifactregistry.googleapis.com when image is from Artifact Registry."
  }
}

# ── Service Account ───────────────────────────────────────────────────────────

run "service_account" {
  command = plan

  assert {
    condition     = google_service_account.ipam.account_id == "ipam-autopilot"
    error_message = "Service account ID should be ipam-autopilot."
  }

  assert {
    condition     = google_service_account.ipam.project == "test-project"
    error_message = "Service account should belong to the specified project."
  }
}

# ── IAM ───────────────────────────────────────────────────────────────────────

run "iam_sql_roles_always_created" {
  command = plan

  assert {
    condition     = google_project_iam_member.sql_client.role == "roles/cloudsql.client"
    error_message = "cloudsql.client IAM role should always be created."
  }

  assert {
    condition     = google_project_iam_member.sql_instance_user.role == "roles/cloudsql.instanceUser"
    error_message = "cloudsql.instanceUser IAM role should always be created."
  }
}

run "cai_iam_created_when_org_id_set" {
  command = plan

  variables {
    organization_id = "123456789012"
  }

  assert {
    condition     = length(google_organization_iam_member.cai_viewer) == 1
    error_message = "Should create CAI viewer IAM member when organization_id is set."
  }

  assert {
    condition     = google_organization_iam_member.cai_viewer[0].role == "roles/cloudasset.viewer"
    error_message = "CAI IAM role should be roles/cloudasset.viewer."
  }
}

run "cai_iam_skipped_when_no_org_id" {
  command = plan

  assert {
    condition     = length(google_organization_iam_member.cai_viewer) == 0
    error_message = "Should not create CAI IAM member when organization_id is empty."
  }
}

# ── Private Service Access ────────────────────────────────────────────────────

# ── Existing Cloud SQL ────────────────────────────────────────────────────────

run "existing_database_skips_mysql_module" {
  command = plan

  variables {
    create_database                   = false
    database_instance_connection_name = "test-project:europe-west1:existing-ipam"
  }

  assert {
    condition     = length(module.mysql) == 0
    error_message = "Should not create Cloud SQL when create_database=false."
  }
}

# ── Backup configuration ─────────────────────────────────────────────────────

run "backup_configuration_uses_defaults" {
  command = plan

  assert {
    condition     = module.mysql[0].instance_name != ""
    error_message = "MySQL instance should be planned with default backup config."
  }
}

run "backup_configuration_custom_values_accepted" {
  command = plan

  variables {
    database_backup_configuration = {
      enabled                        = false
      binary_log_enabled             = false
      start_time                     = "03:00"
      retained_backups               = 14
      retention_unit                 = "COUNT"
      transaction_log_retention_days = "7"
    }
  }

  assert {
    condition     = module.mysql[0].instance_name != ""
    error_message = "MySQL instance should be planned with custom backup config."
  }
}

# ── Private Service Access ────────────────────────────────────────────────────

run "private_service_access_created_with_private_ip" {
  command = plan

  variables {
    cloud_sql_private_ip = true
  }

  assert {
    condition     = length(module.private_service_access) == 1
    error_message = "Should create private service access when cloud_sql_private_ip=true."
  }
}

run "private_service_access_skipped_without_private_ip" {
  command = plan

  variables {
    cloud_sql_private_ip = false
  }

  assert {
    condition     = length(module.private_service_access) == 0
    error_message = "Should skip private service access when cloud_sql_private_ip=false."
  }
}
