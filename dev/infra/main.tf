module "infra" {
  source = "github.com/hetznercloud/kubernetes-dev-env//modules/infra?ref=v0.11.0"

  name             = "hccm-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  worker_count     = 1
  use_cloud_routes = !var.robot_enabled

  hcloud_token = var.hcloud_token

  k3s_channel = var.k3s_channel

  # Share the generated files with the k8s state
  output_dir = abspath("${path.root}/../files")
}
