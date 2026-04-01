# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

variable "ipam_url" {
  description = "URL of the deployed IPAM Autopilot Cloud Run service (from examples/infra output)."
  type        = string
}

variable "project_id" {
  description = "GCP project ID to create the VPC and subnets in."
  type        = string
}

variable "region" {
  description = "GCP region for the subnets."
  type        = string
  default     = "europe-west1"
}
