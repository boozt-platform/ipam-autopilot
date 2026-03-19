# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Local development example — requires docker compose up to be running.
#
# Setup:
#   1. docker compose up --build -d
#   2. cd provider && go build -o terraform-provider-ipam-autopilot .
#   3. Add dev_overrides to ~/.terraformrc (see below)
#   4. export GCP_IDENTITY_TOKEN=localdev
#   5. terraform plan / apply (no init needed with dev_overrides)
#
# ~/.terraformrc:
#   provider_installation {
#     dev_overrides {
#       "boozt-platform/ipam-autopilot" = "/path/to/repo/provider"
#     }
#     direct {}
#   }

terraform {
  required_providers {
    ipam = {
      source = "boozt-platform/ipam-autopilot"
    }
  }
}

provider "ipam" {
  url = "http://localhost:8080"
}

resource "ipam_routing_domain" "local" {
  name = "local-dev"
  vpcs = []
}

resource "ipam_ip_range" "parent" {
  name       = "local-parent"
  range_size = 8
  cidr       = "10.0.0.0/8"
  domain     = ipam_routing_domain.local.id
}

resource "ipam_ip_range" "gke_nodes" {
  name       = "local-gke-nodes"
  range_size = 22
  parent     = ipam_ip_range.parent.cidr
  domain     = ipam_routing_domain.local.id
  labels = {
    env     = "local"
    purpose = "gke-nodes"
  }
}

output "gke_nodes_cidr" {
  value = ipam_ip_range.gke_nodes.cidr
}
