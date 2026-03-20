# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

output "root_cidr" {
  description = "Allocated root CIDR block for the sandbox domain."
  value       = module.root.cidr
}

output "gke_nodes_cidr" {
  description = "Allocated CIDR for GKE nodes."
  value       = module.gke_nodes.cidr
}

output "gke_pods_cidr" {
  description = "Allocated CIDR for GKE pods."
  value       = module.gke_pods.cidr
}
