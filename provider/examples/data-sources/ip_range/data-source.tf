data "ipam_ip_range" "gke_nodes" {
  name = "prod-gke-nodes"
}

output "cidr" {
  value = data.ipam_ip_range.gke_nodes.cidr
}
