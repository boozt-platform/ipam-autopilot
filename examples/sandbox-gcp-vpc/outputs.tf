# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

output "project_id" {
  description = "Created GCP project ID."
  value       = google_project.sandbox.project_id
}

output "ipam_url" {
  description = "IPAM Autopilot Cloud Run service URL."
  value       = module.ipam.service_url
}

