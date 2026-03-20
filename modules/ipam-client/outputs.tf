# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

output "cidr" {
  description = "Allocated CIDR block (e.g. \"10.0.4.0/22\")."
  value       = ipam_ip_range.this.cidr
}

output "id" {
  description = "IPAM range ID."
  value       = ipam_ip_range.this.id
}
