# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# Prerequisites:
#   - IPAM backend deployed (see examples/sandbox-gcp-vpc)
#   - gcloud auth application-default login

terraform {
  required_version = ">= 1.7"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.26"
    }
    ipam = {
      source  = "boozt-platform/ipam-autopilot"
      version = "~> 1.15"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "ipam" {
  url = var.ipam_url
}
