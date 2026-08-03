# Configured from the infra state, and never from an attribute of a resource
# managed in this state (see the k8s module).
provider "kubernetes" {
  config_path = data.terraform_remote_state.infra.outputs.kubeconfig_filename
}

resource "kubernetes_secret_v1" "robot_credentials" {
  count = var.robot_enabled ? 1 : 0

  metadata {
    name      = "robot"
    namespace = "kube-system"
  }

  data = {
    robot-user     = var.robot_user
    robot-password = var.robot_password
  }
}
