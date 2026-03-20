resource "ipam_routing_domain" "prod" {
  name = "prod"
}

resource "ipam_ip_range" "root" {
  name       = "prod-root"
  range_size = 16
  domain     = ipam_routing_domain.prod.id
}

resource "ipam_ip_range" "gke_nodes" {
  name       = "prod-gke-nodes"
  range_size = 22
  domain     = ipam_routing_domain.prod.id
  parent     = ipam_ip_range.root.cidr
  labels = {
    env     = "prod"
    purpose = "gke-nodes"
  }
}

output "gke_nodes_cidr" {
  value = ipam_ip_range.gke_nodes.cidr
}
