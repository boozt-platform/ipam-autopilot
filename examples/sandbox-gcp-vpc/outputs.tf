# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

output "domain" {
  description = "IPAM routing domain details."
  value       = module.sandbox_network.domain
}

output "networks" {
  description = "IPAM-allocated network blocks with their CIDRs."
  value       = module.sandbox_network.networks
}

output "subnets" {
  description = "Created GCP subnets keyed by network name."
  value = {
    for k, s in google_compute_subnetwork.networks : k => {
      name       = s.name
      cidr       = s.ip_cidr_range
      region     = s.region
      self_link  = s.self_link
    }
  }
}
