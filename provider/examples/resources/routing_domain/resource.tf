resource "ipam_routing_domain" "prod" {
  name = "prod"
  vpcs = ["prod-vpc", "shared-vpc"]
}
