# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

output "cloud_run_url" {
  description = "IPAM Autopilot Cloud Run service URL."
  value       = google_cloud_run_v2_service.ipam.uri
}

output "service_account_email" {
  description = "Service account email used by the IPAM Autopilot service."
  value       = google_service_account.ipam.email
}

output "database_instance_connection_name" {
  description = "Cloud SQL instance connection name (project:region:instance)."
  value       = local.db_instance
}
