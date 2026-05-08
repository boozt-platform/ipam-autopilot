# ipam-infra

Deploys the IPAM Autopilot backend to GCP — Cloud Run service, Cloud SQL (MySQL) with IAM authentication via Cloud SQL Auth Proxy sidecar, and all required IAM bindings.

## Usage

```hcl
module "ipam" {
  source = "github.com/boozt-platform/ipam-autopilot//modules/ipam-infra?ref=v1.15.0"

  project_id = "my-project"
  region     = "europe-west1"
}

output "ipam_url" {
  value = module.ipam.cloud_run_url
}
```

## Post-deploy database setup

The module automatically grants the minimum required MySQL privileges to the IPAM service account via a one-off Cloud Run Job (`<cloud_run_name>-db-grant`). This job runs during `tofu apply` via a `local-exec` provisioner after the Cloud SQL instance and the IAM user are created.

The GRANT job uses a dedicated `ipam-db-setup` service account that reads the `default` MySQL superuser password from Secret Manager. The IPAM application service account never has superuser credentials.

The job applies the following minimum privileges required for schema migrations and runtime:

```sql
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, DROP, INDEX, LOCK TABLES, REFERENCES ON ipam.* TO 'ipam-autopilot'@'%';
```

### Requirements for the Terraform executor

The machine (or CI/CD runner) running `tofu apply` must have:

- `gcloud` CLI installed and authenticated as a principal with `roles/run.developer` (or `run.jobs.run`) on the project
- Network access to the Cloud Run Jobs API (`run.googleapis.com`)

### Re-running the GRANT

The job re-runs automatically when the privilege set changes — the GRANT SQL is hashed into `triggers_replace` on a `terraform_data` resource. Updating to a new module version that adds privileges triggers a re-run on the next `tofu apply`.

To force a re-run manually without changing the GRANT:

```bash
tofu apply -replace='module.ipam.terraform_data.run_db_grant[0]'
```

## Destroying infrastructure with cloud_sql_private_ip = true

When `cloud_sql_private_ip = true`, Cloud Run uses Direct VPC egress, which causes GCP to create a `serverless-ipv4-*` internal address reservation in the subnetwork. GCP releases this reservation asynchronously — per GCP documentation this can take up to 1-2 hours after the Cloud Run service is deleted. If you attempt to delete the subnetwork within that window, Terraform will fail with `resourceInUseByAnotherResource`.

This only affects configurations where the subnetwork is in the same Terraform state as the module. The recommended pattern for production is to manage networking (VPC, subnets) in a separate state from the IPAM module — in that case, `tofu destroy` on the module does not touch the subnet and the issue does not arise.

For sandbox or single-state setups, the safest teardown is to delete the entire project:

```bash
gcloud projects delete <project-id>
```

Project deletion bypasses all resource-level reservation checks and completes immediately.

<!-- BEGIN_TF_DOCS -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cloud_run_allow_unauthenticated"></a> [cloud\_run\_allow\_unauthenticated](#input\_cloud\_run\_allow\_unauthenticated) | Allow unauthenticated (public) access to the Cloud Run service. Enable only for testing/sandbox; production should use IAM-authenticated callers. | `bool` | `false` | no |
| <a name="input_cloud_run_deletion_protection"></a> [cloud\_run\_deletion\_protection](#input\_cloud\_run\_deletion\_protection) | Enable deletion protection on the Cloud Run service. | `bool` | `false` | no |
| <a name="input_cloud_run_direct_vpc"></a> [cloud\_run\_direct\_vpc](#input\_cloud\_run\_direct\_vpc) | Connect Cloud Run to the VPC using Direct VPC egress. Required when the Cloud SQL instance is on a private IP. Only relevant when create\_database = false; when create\_database = true the database is always private and Direct VPC is always enabled. | `bool` | `true` | no |
| <a name="input_cloud_run_ingress"></a> [cloud\_run\_ingress](#input\_cloud\_run\_ingress) | Cloud Run ingress setting. One of: INGRESS\_TRAFFIC\_ALL, INGRESS\_TRAFFIC\_INTERNAL\_ONLY, INGRESS\_TRAFFIC\_INTERNAL\_LOAD\_BALANCER. | `string` | `"INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"` | no |
| <a name="input_cloud_run_max_instances"></a> [cloud\_run\_max\_instances](#input\_cloud\_run\_max\_instances) | Maximum number of Cloud Run instances. | `number` | `10` | no |
| <a name="input_cloud_run_name"></a> [cloud\_run\_name](#input\_cloud\_run\_name) | Name for the Cloud Run service. | `string` | `"ipam"` | no |
| <a name="input_cloud_sql_proxy_image"></a> [cloud\_sql\_proxy\_image](#input\_cloud\_sql\_proxy\_image) | Cloud SQL Auth Proxy container image. Pin to a specific version for production stability. | `string` | `"gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.21.2"` | no |
| <a name="input_create_database"></a> [create\_database](#input\_create\_database) | Whether to create a new Cloud SQL instance. Set to false to use an existing instance via database\_instance\_connection\_name. | `bool` | `true` | no |
| <a name="input_database_backup_configuration"></a> [database\_backup\_configuration](#input\_database\_backup\_configuration) | Cloud SQL backup configuration for the IPAM database. | <pre>object({<br/>    enabled                        = optional(bool, true)<br/>    binary_log_enabled             = optional(bool, true)<br/>    start_time                     = optional(string, "02:00")<br/>    location                       = optional(string, null)<br/>    transaction_log_retention_days = optional(string, "7")<br/>    retained_backups               = optional(number, 14)<br/>    retention_unit                 = optional(string, "COUNT")<br/>  })</pre> | `{}` | no |
| <a name="input_database_deletion_protection"></a> [database\_deletion\_protection](#input\_database\_deletion\_protection) | Enable deletion protection on the Cloud SQL instance. | `bool` | `true` | no |
| <a name="input_database_edition"></a> [database\_edition](#input\_database\_edition) | Cloud SQL edition: ENTERPRISE or ENTERPRISE\_PLUS. ENTERPRISE supports db-custom-* tiers; ENTERPRISE\_PLUS requires db-perf-optimized-N-* tiers. | `string` | `"ENTERPRISE"` | no |
| <a name="input_database_instance_connection_name"></a> [database\_instance\_connection\_name](#input\_database\_instance\_connection\_name) | Existing Cloud SQL instance connection name (project:region:instance). Required when create\_database = false. | `string` | `null` | no |
| <a name="input_database_instance_name"></a> [database\_instance\_name](#input\_database\_instance\_name) | Name for the Cloud SQL instance. | `string` | `"ipam-mysql"` | no |
| <a name="input_database_name"></a> [database\_name](#input\_database\_name) | MySQL database name. | `string` | `"ipam"` | no |
| <a name="input_database_tier"></a> [database\_tier](#input\_database\_tier) | Cloud SQL machine tier (e.g. db-f1-micro, db-custom-2-3840, db-perf-optimized-N-2). | `string` | `"db-f1-micro"` | no |
| <a name="input_database_version"></a> [database\_version](#input\_database\_version) | MySQL version for the Cloud SQL instance (e.g. MYSQL\_8\_0, MYSQL\_8\_4). | `string` | `"MYSQL_8_4"` | no |
| <a name="input_db_collation"></a> [db\_collation](#input\_db\_collation) | The collation value. | `string` | `"utf8mb3_general_ci"` | no |
| <a name="input_disable_database_migration"></a> [disable\_database\_migration](#input\_disable\_database\_migration) | Set to true to skip automatic database migration on startup. | `bool` | `false` | no |
| <a name="input_image"></a> [image](#input\_image) | Container image for the IPAM Autopilot backend. | `string` | `"ghcr.io/boozt-platform/ipam-autopilot:latest"` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to all resources (Cloud Run service, Cloud SQL instance). | `map(string)` | `{}` | no |
| <a name="input_module_depends_on"></a> [module\_depends\_on](#input\_module\_depends\_on) | (Optional) A list of external resources the module depends\_on. | `any` | `[]` | no |
| <a name="input_module_enabled"></a> [module\_enabled](#input\_module\_enabled) | (Optional) Whether to create resources within the module or not. | `bool` | `true` | no |
| <a name="input_network"></a> [network](#input\_network) | VPC network name or self\_link to use. Defaults to the project's default VPC. | `string` | `"default"` | no |
| <a name="input_organization_id"></a> [organization\_id](#input\_organization\_id) | GCP organization ID used for Cloud Asset Inventory integration (IPAM\_CAI\_ORG\_ID). Leave empty to disable CAI. | `string` | `""` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | GCP project ID to deploy IPAM Autopilot into. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | GCP region for all resources. | `string` | `"europe-west1"` | no |
| <a name="input_subnetwork"></a> [subnetwork](#input\_subnetwork) | VPC subnetwork name to use for Cloud Run VPC access. Defaults to the network name when not set. | `string` | `null` | no |
| <a name="input_zone"></a> [zone](#input\_zone) | GCP zone for the Cloud SQL instance (e.g. europe-west1-b). | `string` | `"europe-west1-b"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_database_instance_connection_name"></a> [database\_instance\_connection\_name](#output\_database\_instance\_connection\_name) | Cloud SQL instance connection name (project:region:instance). |
| <a name="output_module_id"></a> [module\_id](#output\_module\_id) | Composite reference to all key module resources. Reference this output to create an explicit dependency on the module completing (e.g. depends\_on = [module.ipam.module\_id]). |
| <a name="output_service_account_email"></a> [service\_account\_email](#output\_service\_account\_email) | Service account email used by the IPAM Autopilot service. |
| <a name="output_service_url"></a> [service\_url](#output\_service\_url) | IPAM Autopilot Cloud Run service URL. |
<!-- END_TF_DOCS -->
