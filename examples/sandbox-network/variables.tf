# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

variable "ipam_url" {
  description = "IPAM Autopilot service URL (from examples/sandbox-gcp-vpc output ipam_url)."
  type        = string
}

variable "project_id" {
  description = "GCP project ID where subnets will be created (from examples/sandbox-gcp-vpc output project_id)."
  type        = string
}

variable "network" {
  description = "VPC network name to create subnets in."
  type        = string
  default     = "sandbox-vpc"
}

variable "region" {
  description = "GCP region for subnets."
  type        = string
  default     = "europe-west1"
}
