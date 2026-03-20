# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

provider "ipam" {
  url = "https://ipam.example.internal"
}

variables {
  name        = "test-gke-nodes"
  range_size  = 22
  parent_cidr = "10.0.0.0/16"
}

# ── Resource inputs ───────────────────────────────────────────────────────────

run "resource_inputs_passed_through" {
  command = plan

  assert {
    condition     = ipam_ip_range.this.name == "test-gke-nodes"
    error_message = "Range name should match input variable."
  }

  assert {
    condition     = ipam_ip_range.this.range_size == 22
    error_message = "Range size should match input variable."
  }

  assert {
    condition     = ipam_ip_range.this.parent == "10.0.0.0/16"
    error_message = "Parent CIDR should match input variable."
  }
}

# ── Labels ────────────────────────────────────────────────────────────────────

run "labels_empty_by_default" {
  command = plan

  assert {
    condition     = ipam_ip_range.this.labels == null || length(ipam_ip_range.this.labels) == 0
    error_message = "Labels should default to null or empty map."
  }
}

run "labels_passed_through" {
  command = plan

  variables {
    labels = {
      env     = "prod"
      team    = "platform"
      purpose = "gke-nodes"
    }
  }

  assert {
    condition     = ipam_ip_range.this.labels["env"] == "prod"
    error_message = "env label should be passed through."
  }

  assert {
    condition     = ipam_ip_range.this.labels["team"] == "platform"
    error_message = "team label should be passed through."
  }

  assert {
    condition     = length(ipam_ip_range.this.labels) == 3
    error_message = "Should have exactly 3 labels."
  }
}
