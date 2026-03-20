# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

variable "name" {
  description = "Unique name for the IP range."
  type        = string

  validation {
    condition     = length(var.name) > 0
    error_message = "name must not be empty."
  }
}

variable "range_size" {
  description = "Prefix length of the IP range to allocate (e.g. 22 for /22)."
  type        = number

  validation {
    condition     = var.range_size >= 1 && var.range_size <= 32
    error_message = "range_size must be between 1 and 32."
  }
}

variable "parent_cidr" {
  description = "Parent CIDR block from which the range is allocated (e.g. \"10.0.0.0/16\")."
  type        = string
  default     = null

  validation {
    condition     = var.parent_cidr == null || can(cidrnetmask(var.parent_cidr))
    error_message = "parent_cidr must be a valid CIDR block (e.g. \"10.0.0.0/16\")."
  }
}

variable "domain" {
  description = "Routing domain ID to allocate the range in. Defaults to the first domain when not set."
  type        = string
  default     = null
}

variable "labels" {
  description = "Labels to attach to the IP range."
  type        = map(string)
  default     = {}
}
