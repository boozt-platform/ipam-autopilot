# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

mock_provider "google" {}

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
      "secretmanager.googleapis.com",
    ])
    error_message = "Should enable 7 required GCP APIs (secretmanager included when create_database=true)."
  }
}

run "enables_required_apis_without_database" {
  command = plan

  variables {
    create_database                   = false
    database_instance_connection_name = "test-project:europe-west1:existing-ipam"
  }

  assert {
    condition = toset(keys(google_project_service.apis)) == toset([
      "iam.googleapis.com",
      "run.googleapis.com",
      "compute.googleapis.com",
      "cloudasset.googleapis.com",
      "sqladmin.googleapis.com",
      "servicenetworking.googleapis.com",
    ])
    error_message = "Should enable 6 APIs (no secretmanager) when create_database=false."
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

# ── Database edition ─────────────────────────────────────────────────────────

run "database_edition_defaults_to_enterprise" {
  command = plan

  assert {
    condition     = module.mysql[0].instance_name != ""
    error_message = "MySQL instance should be planned with default ENTERPRISE edition."
  }
}

run "database_edition_enterprise_plus_accepted" {
  command = plan

  variables {
    database_edition = "ENTERPRISE_PLUS"
  }

  assert {
    condition     = module.mysql[0].instance_name != ""
    error_message = "MySQL instance should be planned with ENTERPRISE_PLUS edition."
  }
}

run "database_edition_invalid_value_rejected" {
  command = plan

  variables {
    database_edition = "INVALID"
  }

  expect_failures = [var.database_edition]
}

# ── Private Service Access ────────────────────────────────────────────────────

run "private_service_access_created_when_database_created" {
  command = plan

  assert {
    condition     = length(google_compute_global_address.private_service_access) == 1
    error_message = "Should create PSA global address when create_database=true."
  }

  assert {
    condition     = length(google_service_networking_connection.private_service_access) == 1
    error_message = "Should create service networking connection when create_database=true."
  }
}

run "private_service_access_skipped_without_database" {
  command = plan

  variables {
    create_database                   = false
    database_instance_connection_name = "test-project:europe-west1:existing-ipam"
  }

  assert {
    condition     = length(google_compute_global_address.private_service_access) == 0
    error_message = "Should skip PSA global address when create_database=false."
  }

  assert {
    condition     = length(google_service_networking_connection.private_service_access) == 0
    error_message = "Should skip service networking connection when create_database=false."
  }
}

# ── Subnetwork ────────────────────────────────────────────────────────────────

run "subnetwork_defaults_to_network_name" {
  command = plan

  assert {
    condition     = google_cloud_run_v2_service.ipam.template[0].vpc_access[0].network_interfaces[0].subnetwork == google_cloud_run_v2_service.ipam.template[0].vpc_access[0].network_interfaces[0].network
    error_message = "subnetwork should default to the network name when not set."
  }
}

run "subnetwork_explicit_value_used" {
  command = plan

  variables {
    subnetwork = "my-custom-subnet"
  }

  assert {
    condition     = google_cloud_run_v2_service.ipam.template[0].vpc_access[0].network_interfaces[0].subnetwork == "my-custom-subnet"
    error_message = "subnetwork should use the explicitly provided value."
  }
}

# ── Database GRANT job ────────────────────────────────────────────────────────

run "db_grant_resources_created_with_database" {
  command = plan

  assert {
    condition     = length(google_service_account.db_setup) == 1
    error_message = "Should create db_setup service account when create_database=true."
  }

  assert {
    condition     = google_service_account.db_setup[0].account_id == "ipam-db-setup"
    error_message = "db_setup service account ID should be ipam-db-setup."
  }

  assert {
    condition     = length(google_secret_manager_secret.db_default_password) == 1
    error_message = "Should create Secret Manager secret when create_database=true."
  }

  assert {
    condition     = length(google_cloud_run_v2_job.db_grant) == 1
    error_message = "Should create db_grant Cloud Run Job when create_database=true."
  }
}

run "db_grant_resources_skipped_without_database" {
  command = plan

  variables {
    create_database                   = false
    database_instance_connection_name = "test-project:europe-west1:existing-ipam"
  }

  assert {
    condition     = length(google_service_account.db_setup) == 0
    error_message = "Should not create db_setup service account when create_database=false."
  }

  assert {
    condition     = length(google_secret_manager_secret.db_default_password) == 0
    error_message = "Should not create Secret Manager secret when create_database=false."
  }

  assert {
    condition     = length(google_cloud_run_v2_job.db_grant) == 0
    error_message = "Should not create db_grant Cloud Run Job when create_database=false."
  }
}

run "db_grant_job_name_includes_cloud_run_name" {
  command = plan

  variables {
    cloud_run_name = "my-ipam"
  }

  assert {
    condition     = google_cloud_run_v2_job.db_grant[0].name == "my-ipam-db-grant"
    error_message = "db_grant job name should be <cloud_run_name>-db-grant."
  }
}

run "db_grant_job_always_uses_vpc_access" {
  command = plan

  variables {
    subnetwork = "my-subnet"
  }

  assert {
    condition     = length(google_cloud_run_v2_job.db_grant[0].template[0].template[0].vpc_access) == 1
    error_message = "db_grant job should always have VPC access (database is always private)."
  }
}

run "cloud_run_no_vpc_access_when_existing_db_public" {
  command = plan

  variables {
    create_database                   = false
    database_instance_connection_name = "test-project:europe-west1:existing-ipam"
    cloud_run_direct_vpc              = false
  }

  assert {
    condition     = length(google_cloud_run_v2_service.ipam.template[0].vpc_access) == 0
    error_message = "Cloud Run should have no VPC access when create_database=false and cloud_run_direct_vpc=false."
  }
}

run "cloud_run_vpc_access_when_existing_db_private" {
  command = plan

  variables {
    create_database                   = false
    database_instance_connection_name = "test-project:europe-west1:existing-ipam"
    cloud_run_direct_vpc              = true
  }

  assert {
    condition     = length(google_cloud_run_v2_service.ipam.template[0].vpc_access) == 1
    error_message = "Cloud Run should have VPC access when create_database=false and cloud_run_direct_vpc=true."
  }
}
