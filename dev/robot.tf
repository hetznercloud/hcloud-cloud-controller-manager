locals {
  ansible_inventory = yamldecode(file("${path.module}/robot/inventory.yml"))
  robot_ipv4        = local.ansible_inventory["all"]["hosts"]["hccm-test0"]["ansible_host"]
}

resource "null_resource" "reset_robot" {
  count = var.robot_enabled ? 1 : 0
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
    command = <<-EOT
      ssh-copy-id \
        -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -i ${module.dev.ssh_public_key_filename} \
        root@${local.robot_ipv4}
    EOT
  }
}

module "registry_robot" {
  count      = var.robot_enabled ? 1 : 0
  depends_on = [null_resource.reset_robot]

  source = "github.com/hetznercloud/kubernetes-dev-env//k3s_registry?ref=v0.6.0"

  server = { id = "0", ipv4_address = local.robot_ipv4 }
}

resource "null_resource" "k3sup_robot" {
  count      = var.robot_enabled ? 1 : 0
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
    // We already use overlayfs for the root file system on the server.
    // This caused an issue with the overlayfs default snapshotter in
    // containerd. `--snapshotter=native` avoids this issue. We have not
    // noticed any negative performance impact from this, as the whole
    // filesystem is only kept in memory.
    command = <<-EOT
      k3sup join \
        --ssh-key='${module.dev.ssh_private_key_filename}' \
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

provider "kubernetes" {
  config_path = module.dev.kubeconfig_filename
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
