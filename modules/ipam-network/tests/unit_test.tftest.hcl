# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

mock_provider "ipam" {
  mock_resource "ipam_routing_domain" {
    defaults = {
      id   = "1"
      name = "test-vpc"
    }
  }

  mock_resource "ipam_ip_range" {
    defaults = {
      id   = "1"
      cidr = "10.0.0.0/8"
    }
  }
}

variables {
  domain = {
    name = "test-vpc"
    cidr = "10.0.0.0/8"
  }
}

# ── Routing domain ────────────────────────────────────────────────────────────

run "creates_routing_domain" {
  command = plan

  assert {
    condition     = ipam_routing_domain.this.name == "test-vpc"
    error_message = "Routing domain name should match domain.name."
  }
}

# ── Root range ────────────────────────────────────────────────────────────────

run "creates_root_range" {
  command = plan

  assert {
    condition     = ipam_ip_range.root.cidr == "10.0.0.0/8"
    error_message = "Root range cidr should match domain.cidr."
  }

  assert {
    condition     = ipam_ip_range.root.name == "test-vpc"
    error_message = "Root range name should match domain.name."
  }
}

# ── Network blocks ────────────────────────────────────────────────────────────

run "creates_network_blocks" {
  command = plan

  variables {
    networks = {
      "tenant"    = { size = 16 }
      "gke-nodes" = { size = 16 }
    }
  }

  assert {
    condition     = length(ipam_ip_range.network) == 2
    error_message = "Should create one ip_range per network entry."
  }
}

run "network_names_prefixed_with_domain" {
  command = plan

  variables {
    networks = {
      "tenant" = { size = 16 }
    }
  }

  assert {
    condition     = ipam_ip_range.network["tenant"].name == "test-vpc-tenant"
    error_message = "Network block name should be prefixed with domain name."
  }
}

run "network_blocks_use_root_as_parent" {
  command = plan

  variables {
    networks = {
      "tenant" = { size = 16 }
    }
  }

  assert {
    condition     = ipam_ip_range.network["tenant"].parent == ipam_ip_range.root.cidr
    error_message = "Network blocks should use the root range as their parent."
  }
}

run "empty_networks_creates_domain_and_root_only" {
  command = plan

  variables {
    networks = {}
  }

  assert {
    condition     = length(ipam_ip_range.network) == 0
    error_message = "No network ip_range resources should be created when networks is empty."
  }

  assert {
    condition     = ipam_routing_domain.this.name == "test-vpc"
    error_message = "Routing domain should still be created when networks is empty."
  }

  assert {
    condition     = ipam_ip_range.root.cidr == "10.0.0.0/8"
    error_message = "Root range should still be created when networks is empty."
  }
}

# ── Labels ────────────────────────────────────────────────────────────────────

run "network_inherits_root_labels_when_not_set" {
  command = plan

  variables {
    labels = { env = "prod" }
    networks = {
      "tenant" = { size = 16 }
    }
  }

  assert {
    condition     = ipam_ip_range.network["tenant"].labels == tomap({ env = "prod" })
    error_message = "Network block should inherit root labels when its own labels are not set."
  }
}

run "network_labels_override_root_labels" {
  command = plan

  variables {
    labels = { env = "prod" }
    networks = {
      "tenant" = {
        size   = 16
        labels = { team = "platform" }
      }
    }
  }

  assert {
    condition     = ipam_ip_range.network["tenant"].labels == tomap({ team = "platform" })
    error_message = "Per-network labels should fully override root labels."
  }
}

run "root_range_gets_module_labels" {
  command = plan

  variables {
    labels = { env = "prod" }
  }

  assert {
    condition     = ipam_ip_range.root.labels == tomap({ env = "prod" })
    error_message = "Root range should get module-level labels."
  }
}

# ── Validation ────────────────────────────────────────────────────────────────

run "invalid_network_size_rejected" {
  command = plan

  variables {
    networks = {
      "bad" = { size = 33 }
    }
  }

  expect_failures = [var.networks]
}

run "invalid_domain_cidr_rejected" {
  command = plan

  variables {
    domain = {
      name = "test-vpc"
      cidr = "not-a-cidr"
    }
  }

  expect_failures = [var.domain]
}
