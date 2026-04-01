# ipam-infra

Deploys the IPAM Autopilot backend to GCP — Cloud Run service, Cloud SQL (MySQL) with IAM authentication via Cloud SQL Auth Proxy sidecar, and all required IAM bindings.

## Usage

```hcl
module "ipam" {
  source = "github.com/boozt-platform/ipam-autopilot//modules/ipam-infra?ref=v1.9.0"

  project_id = "my-project"
  region     = "europe-west1"
}

output "ipam_url" {
  value = module.ipam.cloud_run_url
}
```

<!-- BEGIN_TF_DOCS -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cloud_run_allow_unauthenticated"></a> [cloud\_run\_allow\_unauthenticated](#input\_cloud\_run\_allow\_unauthenticated) | Allow unauthenticated (public) access to the Cloud Run service. Enable only for testing/sandbox; production should use IAM-authenticated callers. | `bool` | `false` | no |
| <a name="input_cloud_run_deletion_protection"></a> [cloud\_run\_deletion\_protection](#input\_cloud\_run\_deletion\_protection) | Enable deletion protection on the Cloud Run service. | `bool` | `false` | no |
| <a name="input_cloud_run_ingress"></a> [cloud\_run\_ingress](#input\_cloud\_run\_ingress) | Cloud Run ingress setting. One of: INGRESS\_TRAFFIC\_ALL, INGRESS\_TRAFFIC\_INTERNAL\_ONLY, INGRESS\_TRAFFIC\_INTERNAL\_LOAD\_BALANCER. | `string` | `"INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"` | no |
| <a name="input_cloud_run_max_instances"></a> [cloud\_run\_max\_instances](#input\_cloud\_run\_max\_instances) | Maximum number of Cloud Run instances. | `number` | `10` | no |
| <a name="input_cloud_run_name"></a> [cloud\_run\_name](#input\_cloud\_run\_name) | Name for the Cloud Run service. | `string` | `"ipam"` | no |
| <a name="input_cloud_sql_private_ip"></a> [cloud\_sql\_private\_ip](#input\_cloud\_sql\_private\_ip) | Use private IP for Cloud SQL. Requires VPC peering with servicenetworking. Recommended for production. | `bool` | `true` | no |
| <a name="input_cloud_sql_proxy_image"></a> [cloud\_sql\_proxy\_image](#input\_cloud\_sql\_proxy\_image) | Cloud SQL Auth Proxy container image. Pin to a specific version for production stability. | `string` | `"gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.21.2"` | no |
| <a name="input_create_database"></a> [create\_database](#input\_create\_database) | Whether to create a new Cloud SQL instance. Set to false to use an existing instance via database\_instance\_connection\_name. | `bool` | `true` | no |
| <a name="input_database_backup_configuration"></a> [database\_backup\_configuration](#input\_database\_backup\_configuration) | Cloud SQL backup configuration for the IPAM database. | <pre>object({<br/>    enabled                        = optional(bool, true)<br/>    binary_log_enabled             = optional(bool, true)<br/>    start_time                     = optional(string, "02:00")<br/>    location                       = optional(string, null)<br/>    transaction_log_retention_days = optional(string, "7")<br/>    retained_backups               = optional(number, 7)<br/>    retention_unit                 = optional(string, "COUNT")<br/>    point_in_time_recovery_enabled = optional(bool, false)<br/>  })</pre> | `{}` | no |
| <a name="input_database_deletion_protection"></a> [database\_deletion\_protection](#input\_database\_deletion\_protection) | Enable deletion protection on the Cloud SQL instance. | `bool` | `true` | no |
| <a name="input_database_instance_connection_name"></a> [database\_instance\_connection\_name](#input\_database\_instance\_connection\_name) | Existing Cloud SQL instance connection name (project:region:instance). Required when create\_database = false. | `string` | `null` | no |
| <a name="input_database_instance_name"></a> [database\_instance\_name](#input\_database\_instance\_name) | Name for the Cloud SQL instance. | `string` | `"ipam-mysql"` | no |
| <a name="input_database_name"></a> [database\_name](#input\_database\_name) | MySQL database name. | `string` | `"ipam"` | no |
| <a name="input_database_tier"></a> [database\_tier](#input\_database\_tier) | Cloud SQL machine tier (e.g. db-f1-micro, db-n1-standard-1). | `string` | `"db-f1-micro"` | no |
| <a name="input_database_version"></a> [database\_version](#input\_database\_version) | MySQL version for the Cloud SQL instance (e.g. MYSQL\_8\_0, MYSQL\_8\_4). | `string` | `"MYSQL_8_4"` | no |
| <a name="input_disable_database_migration"></a> [disable\_database\_migration](#input\_disable\_database\_migration) | Set to true to skip automatic database migration on startup. | `bool` | `false` | no |
| <a name="input_image"></a> [image](#input\_image) | Container image for the IPAM Autopilot backend. | `string` | `"ghcr.io/boozt-platform/ipam-autopilot:latest"` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to all resources (Cloud Run service, Cloud SQL instance). | `map(string)` | `{}` | no |
| <a name="input_network"></a> [network](#input\_network) | VPC network name or self\_link to use. Defaults to the project's default VPC. | `string` | `"default"` | no |
| <a name="input_organization_id"></a> [organization\_id](#input\_organization\_id) | GCP organization ID used for Cloud Asset Inventory integration (IPAM\_CAI\_ORG\_ID). Leave empty to disable CAI. | `string` | `""` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | GCP project ID to deploy IPAM Autopilot into. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | GCP region for all resources. | `string` | `"europe-west1"` | no |
| <a name="input_zone"></a> [zone](#input\_zone) | GCP zone for the Cloud SQL instance (e.g. europe-west1-b). | `string` | `"europe-west1-b"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_cloud_run_url"></a> [cloud\_run\_url](#output\_cloud\_run\_url) | IPAM Autopilot Cloud Run service URL. |
| <a name="output_database_instance_connection_name"></a> [database\_instance\_connection\_name](#output\_database\_instance\_connection\_name) | Cloud SQL instance connection name (project:region:instance). |
| <a name="output_service_account_email"></a> [service\_account\_email](#output\_service\_account\_email) | Service account email used by the IPAM Autopilot service. |
<!-- END_TF_DOCS -->
