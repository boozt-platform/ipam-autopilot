# ipam-network

Registers a VPC network and its top-level IP blocks in IPAM Autopilot. Creates a routing domain representing the network and allocates the defined CIDR blocks as parent ranges from which tenant subnets can be carved out.

## Usage

```hcl
module "prod_network" {
  source = "github.com/boozt-platform/ipam-autopilot//modules/ipam-network?ref=v1.9.0"

  domain = "prod-vpc"
  labels = { env = "prod" }

  networks = {
    "tenant" = {
      size   = 16
      labels = { purpose = "tenant-workloads" }
    }
    "gke-nodes" = {
      size   = 16
      labels = { purpose = "gke-nodes" }
    }
  }
}

# Allocate a tenant subnet from the tenant block
resource "ipam_ip_range" "my_team" {
  name   = "my-team-prod"
  size   = 22
  domain = module.prod_network.domain_id
  parent = module.prod_network.networks["tenant"].cidr
}
```

<!-- BEGIN_TF_DOCS -->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_domain"></a> [domain](#input\_domain) | Name of the routing domain (e.g. "prod-vpc"). Represents the VPC network in IPAM. | `string` | n/a | yes |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels applied to all network blocks. Merged with per-block labels. | `map(string)` | `{}` | no |
| <a name="input_networks"></a> [networks](#input\_networks) | Map of top-level network blocks to register. Key is the block name, value contains the prefix size and optional labels. | <pre>map(object({<br/>    size   = number<br/>    labels = optional(map(string), {})<br/>  }))</pre> | `{}` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_domain_id"></a> [domain\_id](#output\_domain\_id) | Routing domain ID. |
| <a name="output_networks"></a> [networks](#output\_networks) | Map of registered network blocks with their allocated CIDRs. |
<!-- END_TF_DOCS -->
