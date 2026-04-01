# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

output "domain" {
  description = "Routing domain details."
  value = {
    id     = ipam_routing_domain.this.id
    name   = ipam_routing_domain.this.name
    cidr   = ipam_ip_range.root.cidr
    labels = ipam_ip_range.root.labels
  }
}

output "networks" {
  description = "Map of registered network blocks with their allocated CIDRs."
  value = {
    for k, v in ipam_ip_range.network : k => {
      id     = v.id
      name   = v.name
      cidr   = v.cidr
      labels = v.labels
    }
  }
}
