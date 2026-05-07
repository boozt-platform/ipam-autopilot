# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

resource "terraform_data" "module_depends_on" {
  input = var.module_depends_on
}

resource "ipam_routing_domain" "this" {
  name = var.domain.name

  depends_on = [terraform_data.module_depends_on]
}

resource "ipam_ip_range" "root" {
  name   = var.domain.name
  cidr   = var.domain.cidr
  domain = ipam_routing_domain.this.id
  labels = var.labels
}

resource "ipam_ip_range" "network" {
  for_each   = var.networks
  name       = "${var.domain.name}-${each.key}"
  range_size = each.value.size
  parent     = ipam_ip_range.root.cidr
  domain     = ipam_routing_domain.this.id
  labels     = each.value.labels != null ? each.value.labels : var.labels
}
