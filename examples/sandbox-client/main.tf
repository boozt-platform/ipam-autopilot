# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# Example: allocate IP ranges from a running IPAM Autopilot instance.
#
# Deploy the IPAM service first:
#   cd ../sandbox && tofu apply
#
# Then apply this example:
#   tofu init
#   tofu apply -var="ipam_url=$(cd ../sandbox && tofu output -raw ipam_url)"

# ── Routing domain ────────────────────────────────────────────────────────────

resource "ipam_routing_domain" "sandbox" {
  name = "sandbox"
}

# ── Root range (the full address space for this domain) ───────────────────────

module "root" {
  source = "../../modules/ipam-client"

  name        = "sandbox-root"
  range_size  = 16
  domain      = ipam_routing_domain.sandbox.id
  labels = {
    env = "sandbox"
  }
}

# ── Child ranges allocated from the root ──────────────────────────────────────

module "gke_nodes" {
  source = "../../modules/ipam-client"

  name        = "sandbox-gke-nodes"
  range_size  = 22
  parent_cidr = module.root.cidr
  domain      = ipam_routing_domain.sandbox.id
  labels = {
    env     = "sandbox"
    purpose = "gke-nodes"
  }
}

module "gke_pods" {
  source = "../../modules/ipam-client"

  name        = "sandbox-gke-pods"
  range_size  = 16
  parent_cidr = module.root.cidr
  domain      = ipam_routing_domain.sandbox.id
  labels = {
    env     = "sandbox"
    purpose = "gke-pods"
  }
}
