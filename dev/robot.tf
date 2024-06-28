terraform {
  required_providers {
    ansible = {
      version = "~> 1.3.0"
      source  = "ansible/ansible"
    }
  }
}

locals {
  ansible_inventory = yamldecode(file("${path.module}/../hack/robot-e2e/inventory.yml"))
  host_vars = local.ansible_inventory["all"]["hosts"]["hccm-test0"]

  robot_ipv4 = local.host_vars["ansible_host"]
}

resource "ansible_playbook" "robot_setup" {
  count = 0

  depends_on = [module.dev]

  name = "hccm-test0"
  playbook = "../hack/robot-e2e/setup.yml"
  verbosity = 2

  extra_vars = merge(local.host_vars, {
    control_server_ipv4 =  module.dev.control_server_ipv4
  })
}

resource "null_resource" "reset_robot" {
  count = var.setup_robot ? 1 : 0
  triggers = {
    # Wait the control-plane to be initialized, and re-join the new cluster if the
    # control-plane server changed.
    control_id = module.dev.control_server_ipv4
  }

  connection {
    host = local.robot_ipv4
  }
  provisioner "remote-exec" {
    inline = [
      # Only reboot if the node was already provisioned since the last reboot
      "stat /etc/rancher/k3s 1>/dev/null && systemctl reboot ; exit 0",
    ]
  }

  provisioner "remote-exec" {
    connection {
      timeout = "3m"
    }

    inline = [
      "whoami"
    ]
  }

  provisioner "local-exec" {
    command = "ssh-copy-id -i ${module.dev.ssh_public_key} root@${local.robot_ipv4}"
  }
}

module "registry_robot" {
  count = var.setup_robot ? 1 : 0
  depends_on = [null_resource.reset_robot]

  source = "../../terraform-k8s-dev/k3s_registry"

  server = { id = "0", ipv4_address = local.robot_ipv4}
  private_key = file(module.dev.ssh_private_key)
}

resource "null_resource" "k3sup_robot" {
  count = var.setup_robot ? 1 : 0
  depends_on = [module.registry_robot.0]

  triggers = {
    # Wait the control-plane to be initialized, and re-join the new cluster if the
    # control-plane server changed.
    control_id = module.dev.control_server_ipv4
  }

  connection {
    host = local.robot_ipv4
  }

   provisioner "local-exec" {
    command = <<-EOT
      k3sup join \
        --ssh-key='${module.dev.ssh_private_key}' \
        --ip='${local.robot_ipv4}' \
        --server-ip='${module.dev.control_server_ipv4}' \
        --k3s-channel='${var.k3s_channel}' \
        --k3s-extra-args="\
          --kubelet-arg='cloud-provider=external' \
          --node-ip='${local.robot_ipv4}' \
          --node-label instance.hetzner.cloud/is-root-server=true \
          --snapshotter=native" \
      EOT
  }
}

data "local_sensitive_file" "kubeconfig" {
  depends_on = [module.dev]
  filename   = "${path.root}/files/kubeconfig.yaml"
}

provider "kubernetes" {
  config_path = data.local_sensitive_file.kubeconfig.filename
}

resource "kubernetes_secret_v1" "hcloud_token" {
  count = var.setup_robot ? 1 : 0

  metadata {
    name      = "hcloud-robot"
    namespace = "kube-system"
  }

  data = {
    robot-user = var.robot_user
    robot-password = var.robot_password
  }
}
