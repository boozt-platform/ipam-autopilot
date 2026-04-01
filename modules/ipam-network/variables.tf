# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

variable "domain" {
  description = "Routing domain definition. name is the domain identifier (e.g. \"prod-vpc\"); cidr is the root address space allocated to it (e.g. \"10.0.0.0/8\")."
  type = object({
    name = string
    cidr = string
  })

  validation {
    condition     = length(var.domain.name) > 0
    error_message = "domain.name must not be empty."
  }

  validation {
    condition     = can(cidrnetmask(var.domain.cidr))
    error_message = "domain.cidr must be a valid CIDR notation (e.g. \"10.0.0.0/8\")."
  }
}

variable "networks" {
  description = "Map of network blocks to carve from the domain's root CIDR. Key is the block name; size is the prefix length. labels overrides the module-level labels when set."
  type = map(object({
    size   = number
    labels = optional(map(string))
  }))
  default = {}

  validation {
    condition     = alltrue([for k, v in var.networks : v.size >= 1 && v.size <= 32])
    error_message = "Each network size must be between 1 and 32."
  }
}

variable "labels" {
  description = "Labels applied to all resources. Per-network labels override these when set."
  type        = map(string)
  default     = {}
}
