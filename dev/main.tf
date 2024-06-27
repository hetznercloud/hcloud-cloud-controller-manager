module "dev" {
  source = "github.com/hetznercloud/terraform-k8s-dev?ref=v0.3.0"

  name         = "hccm-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  worker_count = 1
  # We deploy hccm through skaffold, its the application under development/test.
  deploy_hccm      = false
  use_cloud_routes = !var.setup_robot

  hcloud_token = var.hcloud_token

  k3s_channel = var.k3s_channel
}
