# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# Prerequisites: run examples/infra first to deploy the IPAM service, then
# set ipam_url to the Cloud Run URL from its output:
#   cd ../sandbox && tofu output ipam_url
#
# Authentication: the provider uses Google Application Default Credentials to
# obtain an identity token with the Cloud Run service URL as audience.
# Run `gcloud auth application-default login` before applying.
# For CI/CD set IPAM_IDENTITY_TOKEN to a valid Google identity token.

terraform {
  required_version = ">= 1.7"
  required_providers {
    ipam = {
      source  = "boozt-platform/ipam-autopilot"
      version = "~> 1.8"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
  }
}

provider "ipam" {
  url = var.ipam_url
}

provider "google" {
  project = var.project_id
  region  = var.region
}
