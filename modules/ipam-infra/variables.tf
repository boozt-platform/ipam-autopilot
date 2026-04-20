# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# ── Project ────────────────────────────────────────────────────────────────────

variable "project_id" {
  description = "GCP project ID to deploy IPAM Autopilot into."
  type        = string
}

variable "region" {
  description = "GCP region for all resources."
  type        = string
  default     = "europe-west1"
}

variable "zone" {
  description = "GCP zone for the Cloud SQL instance (e.g. europe-west1-b)."
  type        = string
  default     = "europe-west1-b"
}

variable "organization_id" {
  description = "GCP organization ID used for Cloud Asset Inventory integration (IPAM_CAI_ORG_ID). Leave empty to disable CAI."
  type        = string
  default     = ""
}

# ── Container image ────────────────────────────────────────────────────────────

variable "image" {
  description = "Container image for the IPAM Autopilot backend."
  type        = string
  default     = "ghcr.io/boozt-platform/ipam-autopilot:latest"
}


# ── Network ────────────────────────────────────────────────────────────────────

variable "network" {
  description = "VPC network name or self_link to use. Defaults to the project's default VPC."
  type        = string
  default     = "default"
}

variable "cloud_sql_private_ip" {
  description = "Use private IP for Cloud SQL. Requires VPC peering with servicenetworking. Recommended for production."
  type        = bool
  default     = true
}


# ── Cloud SQL ──────────────────────────────────────────────────────────────────

variable "create_database" {
  description = "Whether to create a new Cloud SQL instance. Set to false to use an existing instance via database_instance_connection_name."
  type        = bool
  default     = true
}

variable "database_instance_connection_name" {
  description = "Existing Cloud SQL instance connection name (project:region:instance). Required when create_database = false."
  type        = string
  default     = null
}

variable "database_version" {
  description = "MySQL version for the Cloud SQL instance (e.g. MYSQL_8_0, MYSQL_8_4)."
  type        = string
  default     = "MYSQL_8_4"

  validation {
    condition     = can(regex("^MYSQL_", var.database_version))
    error_message = "database_version must be a MySQL version string (e.g. MYSQL_8_4)."
  }
}

variable "database_instance_name" {
  description = "Name for the Cloud SQL instance."
  type        = string
  default     = "ipam-mysql"
}

variable "database_tier" {
  description = "Cloud SQL machine tier (e.g. db-f1-micro, db-n1-standard-1)."
  type        = string
  default     = "db-f1-micro"
}

variable "database_name" {
  description = "MySQL database name."
  type        = string
  default     = "ipam"
}

variable "database_deletion_protection" {
  description = "Enable deletion protection on the Cloud SQL instance."
  type        = bool
  default     = true
}

variable "db_collation" {
  description = "The collation value."
  type        = string
  default     = "utf8mb3_general_ci"
}

# ── Cloud Run ──────────────────────────────────────────────────────────────────

variable "cloud_run_name" {
  description = "Name for the Cloud Run service."
  type        = string
  default     = "ipam"
}

variable "cloud_run_max_instances" {
  description = "Maximum number of Cloud Run instances."
  type        = number
  default     = 10
}

variable "cloud_run_deletion_protection" {
  description = "Enable deletion protection on the Cloud Run service."
  type        = bool
  default     = false
}

variable "cloud_run_allow_unauthenticated" {
  description = "Allow unauthenticated (public) access to the Cloud Run service. Enable only for testing/sandbox; production should use IAM-authenticated callers."
  type        = bool
  default     = false
}

variable "cloud_run_ingress" {
  description = "Cloud Run ingress setting. One of: INGRESS_TRAFFIC_ALL, INGRESS_TRAFFIC_INTERNAL_ONLY, INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER."
  type        = string
  default     = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  validation {
    condition     = contains(["INGRESS_TRAFFIC_ALL", "INGRESS_TRAFFIC_INTERNAL_ONLY", "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"], var.cloud_run_ingress)
    error_message = "cloud_run_ingress must be one of: INGRESS_TRAFFIC_ALL, INGRESS_TRAFFIC_INTERNAL_ONLY, INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER."
  }
}

variable "cloud_sql_proxy_image" {
  description = "Cloud SQL Auth Proxy container image. Pin to a specific version for production stability."
  type        = string
  default     = "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.21.2"
}

variable "database_backup_configuration" {
  description = "Cloud SQL backup configuration for the IPAM database."
  type = object({
    enabled                        = optional(bool, true)
    binary_log_enabled             = optional(bool, true)
    start_time                     = optional(string, "02:00")
    location                       = optional(string, null)
    transaction_log_retention_days = optional(string, "7")
    retained_backups               = optional(number, 14)
    retention_unit                 = optional(string, "COUNT")
  })
  default = {}

  validation {
    condition = (
      !var.database_backup_configuration.enabled ||
      var.database_backup_configuration.retained_backups > tonumber(var.database_backup_configuration.transaction_log_retention_days)
    )
    error_message = "retained_backups must be greater than transaction_log_retention_days when backup is enabled."
  }
}

variable "disable_database_migration" {
  description = "Set to true to skip automatic database migration on startup."
  type        = bool
  default     = false
}

# ── Labels ─────────────────────────────────────────────────────────────────────

variable "labels" {
  description = "Labels to apply to all resources (Cloud Run service, Cloud SQL instance)."
  type        = map(string)
  default     = {}
}
