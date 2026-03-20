# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

resource "ipam_ip_range" "this" {
  name       = var.name
  range_size = var.range_size
  parent     = var.parent_cidr
  domain     = var.domain
  labels     = var.labels
}
