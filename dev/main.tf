module "dev" {
  source = "../../terraform-k8s-dev"

  name         = "hccm-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  worker_count = 1
  # We deploy hccm through skaffold, its the application under development/test.
  enable_hccm  = false
  enable_hccm_routes = !var.setup_robot

  hcloud_token = var.hcloud_token

  k3s_channel = var.k3s_channel
}

