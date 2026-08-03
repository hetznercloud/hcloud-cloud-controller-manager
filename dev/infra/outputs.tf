# Consumed by the k8s state through the terraform_remote_state data source.

output "kubeconfig_filename" {
  description = "Path to the Kubeconfig file"
  value       = module.infra.kubeconfig_filename
}

output "network_id" {
  description = "ID of the Hetzner Cloud Network the cluster runs in"
  value       = module.infra.network_id
}

output "cluster_cidr" {
  description = "CIDR range for the Pods"
  value       = module.infra.cluster_cidr
}

output "use_cloud_routes" {
  description = "Whether the Hetzner Cloud network routes are used for Pod traffic"
  value       = module.infra.use_cloud_routes
}

output "registry_service_ip" {
  description = "ClusterIP of the in-cluster Docker registry service"
  value       = module.infra.registry_service_ip
}
