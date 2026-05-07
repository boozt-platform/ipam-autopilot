# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

variable "billing_account" {
  description = "GCP billing account ID to associate with the new project (e.g. 012345-ABCDEF-012345)."
  type        = string
}

variable "org_id" {
  description = "GCP organization ID under which the sandbox project will be created."
  type        = string
}

variable "region" {
  description = "GCP region for all resources."
  type        = string
  default     = "europe-west1"
}

variable "zone" {
  description = "GCP zone for the Cloud SQL instance."
  type        = string
  default     = "europe-west1-b"
}
