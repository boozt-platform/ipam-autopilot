# IPAM Autopilot

> **Fork notice:** This is a Boozt Fashion AB fork of
> [GoogleCloudPlatform/professional-services](https://github.com/GoogleCloudPlatform/professional-services/tree/main/tools/ipam-autopilot),
> maintained at [boozt-platform/ipam-autopilot](https://github.com/boozt-platform/ipam-autopilot).
> See [NOTICE](./NOTICE) and [CHANGES.md](./CHANGES.md) for details.

IPAM Autopilot is a simple Docker Container and Terraform provider that allows you to automatically manage IP ranges for GCP VPCs.

It connects to Cloud Asset Inventory to also retrieve already existing subnets and ranges, in order to allow for a mixed usage.

![Architecture showing Terraform, CloudRun, CloudSQL and Cloud Asset Inventory](./img/architecture.png "IPAM Autopilot Architecture")

IPAM Autopilot consists of two parts, a [backend service](./container) that provides a Rest API and a [terraform provider](./provider).

The provider uses application default credentials to authenticate against the backend. Alternatively you can directly provide an identity token via the `GCP_IDENTITY_TOKEN` environment variable to access the Cloud Run instance, the audience for the identity token should be the domain of the Cloud Run service.

## Local development

### Prerequisites
- Docker + Docker Compose
- Go 1.26
- Terraform or OpenTofu

Alternatively, open the repo in VS Code — it includes a [devcontainer](./.devcontainer) with all tools pre-installed (Go 1.26, golangci-lint, docker-in-docker).

### Start the stack

```bash
docker compose up --build -d   # MySQL + API (localhost:8080) + Jaeger (localhost:16686)
```

### Common tasks

```bash
make test                # unit tests (container + provider)
make test-integration    # integration tests via testcontainers (requires Docker)
make lint                # golangci-lint
make fmt                 # gofmt

make dev-apply           # build provider binary + terraform apply against docker-compose
make dev-destroy         # terraform destroy local dev resources
```

### Environment variables

All environment variables are prefixed with `IPAM_`, except OpenTelemetry standard variables.

**Database**

| Variable | Default | Description |
|---|---|---|
| `IPAM_DATABASE_USER` | | MySQL username |
| `IPAM_DATABASE_PASSWORD` | | MySQL password |
| `IPAM_DATABASE_HOST` | | MySQL `host:port` |
| `IPAM_DATABASE_NAME` | | MySQL database name |
| `IPAM_DATABASE_NET` | `tcp` | MySQL network type |
| `IPAM_DISABLE_DATABASE_MIGRATION` | `FALSE` | Set `TRUE` to skip auto-migration on startup |

**Server**

| Variable | Default | Description |
|---|---|---|
| `IPAM_PORT` | `8080` | HTTP listen port |
| `IPAM_LOG_FORMAT` | `json` | Set `text` for human-readable logs |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | | OTLP gRPC endpoint for tracing (e.g. `jaeger:4317`) — standard OpenTelemetry variable |

**Features**

| Variable | Default | Description |
|---|---|---|
| `IPAM_CAI_ORG_ID` | | GCP organisation ID for Cloud Asset Inventory integration (see [CAI integration](#cloud-asset-inventory-integration)) |
| `IPAM_CAI_DB_SYNC` | `FALSE` | Set `TRUE` to enable DB-backed CAI cache with background sync; default queries CAI live on each allocation |
| `IPAM_CAI_SYNC_INTERVAL` | `5m` | Sync interval when `IPAM_CAI_DB_SYNC=TRUE` (Go duration, e.g. `5m`, `1h`) |
| `IPAM_DISABLE_BULK_IMPORT` | `FALSE` | Set `TRUE` to disable `POST /api/v1/ranges/import` |
| `IPAM_STORAGE_BUCKET` | | GCS bucket name for the legacy built-in provider registry |

**Terraform provider**

| Variable | Default | Description |
|---|---|---|
| `IPAM_URL` | | IPAM API base URL (alternative to `url` in provider config) |
| `IPAM_IDENTITY_TOKEN` | | GCP identity token for authenticating against Cloud Run; if unset, Application Default Credentials are used automatically |

---

## API

All IPAM endpoints are available under `/api/v1`:

```
POST   /api/v1/ranges              allocate a new IP range (auto or direct CIDR)
GET    /api/v1/ranges              list all ranges; optional ?name= filter
GET    /api/v1/ranges/:id          get a single range
DELETE /api/v1/ranges/:id          release a range
POST   /api/v1/ranges/import       bulk-import pre-existing CIDRs (idempotent)

POST   /api/v1/domains             create a routing domain
GET    /api/v1/domains             list all routing domains
GET    /api/v1/domains/:id         get a single routing domain
PUT    /api/v1/domains/:id         update a routing domain
DELETE /api/v1/domains/:id         delete a routing domain

GET    /api/v1/audit?limit=N       last N audit log events (default 100, max 1000)
```

Legacy paths (`/ranges`, `/domains`) are kept for Terraform provider backward compatibility.

---

## Cloud Asset Inventory integration

When `IPAM_CAI_ORG_ID` is set, IPAM Autopilot uses [Cloud Asset Inventory](https://cloud.google.com/asset-inventory) to discover existing VPC subnets across your GCP organisation and merge them with IPAM DB allocations before the collision avoidance algorithm runs. This prevents double-allocation even for subnets that exist in GCP but were never registered in IPAM.

**VPC scoping** — Each routing domain has an optional list of VPCs. Only subnets belonging to the domain's VPCs are considered. This means IPAM is safe to use across multiple independent VPCs: allocating a `/22` for `prod-vpc` will not be blocked by subnets in `staging-vpc` if they are in separate routing domains.

**VPC name format** — The VPC list in a routing domain accepts both full resource URLs and short names:

```
# Full URL (as returned by gcloud)
https://www.googleapis.com/compute/v1/projects/my-project/global/networks/my-vpc

# Short name — simpler, works the same way
my-vpc
```

**Two modes**

| Mode | Config | Behaviour |
|---|---|---|
| Live (default) | `IPAM_CAI_ORG_ID` only | CAI API is queried on every `POST /ranges` with `range_size`; always up to date, adds latency per allocation |
| DB sync | `IPAM_CAI_ORG_ID` + `IPAM_CAI_DB_SYNC=TRUE` | Subnets are synced into a local `cai_subnets` table on startup and refreshed every `IPAM_CAI_SYNC_INTERVAL` (default `5m`) in the background; allocations read from DB — fast, no per-request CAI latency |

Use **DB sync mode** when you have many allocations per minute or want to reduce CAI API calls. Use **live mode** when you need guaranteed up-to-date collision avoidance and allocation frequency is low.

**Required IAM** — The Cloud Run service account needs `roles/cloudasset.viewer` at the organisation level.

---

## Bulk import

Use `POST /api/v1/ranges/import` to register pre-existing CIDRs in IPAM without triggering auto-allocation. This is the recommended first step when adopting IPAM for a VPC that already has manually-assigned subnets.

### When to use

- Migrating an existing VPC into IPAM management
- Registering subnets that were created outside of Terraform so that future auto-allocations do not overlap with them
- The endpoint is a complement to CAI integration: CAI prevents overlaps at allocation time, while bulk import makes subnets visible in IPAM for `terraform import` and audit purposes

### Request

`POST /api/v1/ranges/import` accepts a JSON array:

```json
[
  {
    "name":   "gke-nodes",
    "cidr":   "10.1.0.0/22",
    "domain": "1",
    "labels": { "env": "prod", "purpose": "gke-nodes" }
  },
  {
    "name":   "gke-pods",
    "cidr":   "10.2.0.0/16",
    "domain": "1"
  }
]
```

| Field | Required | Description |
|---|---|---|
| `name` | yes | Range name (max 255 characters) |
| `cidr` | yes | CIDR block to register (e.g. `10.1.0.0/22`) |
| `domain` | no | Routing domain ID; defaults to the first domain |
| `parent` | no | Parent range ID |
| `labels` | no | Key/value metadata (key ≤ 63 chars, value ≤ 255 chars) |

The endpoint is **idempotent**: ranges with the same CIDR already present in the same domain are silently skipped. Each imported range is written to the audit log.

### Response

```json
{ "imported": 2, "skipped": 0, "errors": [] }
```

### Workflow: adopting an existing VPC

1. **List existing subnets** in the VPC:

   ```bash
   gcloud compute networks subnets list \
     --filter="network=projects/MY_PROJECT/global/networks/MY_VPC" \
     --format="value(name,ipCidrRange)"
   ```

2. **Register them in IPAM**:

   ```bash
   curl -X POST https://YOUR_IPAM_URL/api/v1/ranges/import \
     -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
     -H "Content-Type: application/json" \
     -d '[
       {"name": "gke-nodes", "cidr": "10.1.0.0/22", "domain": "1"},
       {"name": "gke-pods",  "cidr": "10.2.0.0/16", "domain": "1"}
     ]'
   ```

3. **Write Terraform resources** for each imported range:

   ```hcl
   resource "ipam_ip_range" "gke_nodes" {
     name   = "gke-nodes"
     cidr   = "10.1.0.0/22"
     domain = "1"
   }

   resource "ipam_ip_range" "gke_pods" {
     name   = "gke-pods"
     cidr   = "10.2.0.0/16"
     domain = "1"
   }
   ```

4. **Import into Terraform state** — get the range IDs from the import response or `GET /api/v1/ranges?name=gke-nodes`:

   ```bash
   terraform import ipam_ip_range.gke_nodes 42
   terraform import ipam_ip_range.gke_pods  43
   terraform plan   # should show no changes
   ```

### Security

To disable the import endpoint in environments where arbitrary CIDR registration is undesirable (e.g. production), set:

```bash
IPAM_DISABLE_BULK_IMPORT=TRUE
```

When disabled, `POST /api/v1/ranges/import` returns `404`.

---

## IPAM Autopilot Backend
The [infrastructure](./infrastructure) folder contains a sample Terraform setup with which the IPAM Autopilot backend can be deployed to CloudRun. The required APIs are created during the deployment. The deployment instructions also provision a small CloudSQL instance as well. The container is build as part of the deployment. The `Dockerfile` containing the build instructions is at the top level, since the container needs files that are generated during the infrastructure deployment.

The following GCP services are used as part of the deployment, and might cause cost:
  * [Cloud Run](https://cloud.google.com/run)
  * [CloudSQL](https://cloud.google.com/sql)
  * [Secret Manager](https://cloud.google.com/secret-manager)
  * [Google Cloud Storage](https://cloud.google.com/storage)

A self-contained registry for discovery of the provider is part of the backend and the deployment. It uses GCS with signed URL for providing the provider binaries. Please be aware, this approach towards Terraform provider registry can be brittle.

You can also disable the automatic database migration using `DISABLE_DATABASE_MIGRATION` if you prefer to do the database migration manually. Therefore you have to set the value to `TRUE`. Or in Terraform use the `disable_database_migration` variable.

## Deploying
In order to use the provider later from terraform we need to provide the providers binaries in a way that Terraform can resolve them.
For this we need to first build the provider binaries. The Terraform deployment instructions will use the binaries to bundle the provider for discovery by the Terraform clients. For this you will need to have PGP set up so that the checksum file that accompanies the binaries can be signed.

The infrastructure deployment takes the following variables, you can either set them via environment variables TF_VAR_<name> or in a .tfvars file.
| Variable                   	| Default Value   	| Description                                                                                                                                                	|
|----------------------------	|-----------------	|------------------------------------------------------------------------------------------------------------------------------------------------------------	|
| organization_id                 	|                 	| ID of the organization, that holds the subnets that are queried via Cloud Asset Inventory.
| project_id                 	|                 	| Project ID of the project to which the IPAM Autopilot should be deployed.                                                                                  	|
| region                     	| europe-west1    	| GCP region that should contain the resources.                                                                                                              	|
| artifact_registry_location 	| europe          	| Location for the Artifact Registry location, containing the Docker container image for the IPAM Autopilot backend.                                         	|
| container_version          	| 1               	| Version of the container, since the container is build by the infrastructure automation, you can use this variable to trigger a new container image build. 	|
| provider_binary_folder     	| ../provider/bin 	| The folder relative to the infrastructure folder containing the binaries of the provider, is generated by `make release`                                   	|
| provider_version           	| 0.1.0           	| Version of the provider, needs to match the version in the Makefile                                                                                        	|
| disable_database_migration           	| FALSE           	| Whether the CloudRun service should automatically migrate the databse |

In order to deploy, you will need to execute the following commands.

1. Build the Terrafrom provider by calling `make release` from the provider folder.
1. Deploy the infrastructure `terraform init` and `terraform apply`

## Setting up the local Terraform setup
In order to be able to download the provider from the CloudRun service, the Terraform CLI will need to authenticate against CloudRun. You will need to setup your `~/.terraformrc` file so that it conaints a valid identity token for a user with `roles/run.invoker`. You can obtain that token with `gcloud auth print-identity-token` and manually create the entry in your terraformrc file.
```
credentials "<cloud run hostname>" {
  token = "<identity token>"
}
```

Or as a condensed command
`echo "credentials \"<cloud run hostname>\" { \n token = \"$(gcloud auth print-identity-token)\" \n }" > ~/.terraformrc`

## An example configuration
Below is a Terraform example for using IPAM Autopilot
```
terraform {
  required_providers {
    ipam = {
      version = "0.1"
      source = "<cloud run hostname>/ipam-autopilot/ipam"
    }
  }
}

provider "ipam" {
  url = "https://<cloud run hostname>"
}

resource "ipam_ip_range" "pod-ranges" {
  range_size = 22
  name = "gke services range"
}

output "range" {
  value = ipam_ip_range.pod-ranges.cidr
}
```
## Subnet selection logic
![IP Subnet selection logic](./img/flow.png "Sequence flow")

A simple example might shed some light on how the selection works. Let's assume we want a `/24` range in the `10.0.0.0/8` subnet. In the IPAM Autopilot database, the subnets `10.0.0.0/28` and `10.0.0.16/28` are allocated. From Cloud Asset Inventory a VPC with a subnet `10.0.0.64/26` is discovered as well. This means that the subnet `10.0.0.0/24` will collide with this subnets, so IPAM Autopilot will allocate `10.0.1.0/24`.