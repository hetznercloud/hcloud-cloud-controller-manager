#cloud-config
write_files:
- content: |
    net.bridge.bridge-nf-call-ip6tables = 1
    net.bridge.bridge-nf-call-iptables = 1
  path: /etc/sysctl.d/k8s.conf
- content: |
    apiVersion: kubeadm.k8s.io/v1beta2
    kind: ClusterConfiguration
    kubernetesVersion: v{{.K8sVersion}}
    networking:
      podSubnet: "10.244.0.0/16"
  path: /tmp/kubeadm-config.yaml
- content: |
    [Service]
    Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
  path: /etc/systemd/system/kubelet.service.d/20-hcloud.conf
- content: |
    alias k="kubectl"
    alias ksy="kubectl -n kube-system"
    alias kgp="kubectl get pods"
    alias kgs="kubectl get services"
    export HCLOUD_TOKEN={{.HcloudToken}}
  path: /root/.bashrc
runcmd:
- export HOME=/root
- sysctl --system
- apt install -y apt-transport-https curl
- curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
- echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list
- apt update
- apt install -y kubectl={{.K8sVersion}}-00 kubeadm={{.K8sVersion}}-00 kubelet={{.K8sVersion}}-00 docker.io
- systemctl daemon-reload
- systemctl restart kubelet
- kubeadm init  --config /tmp/kubeadm-config.yaml
- mkdir -p /root/.kube
- cp -i /etc/kubernetes/admin.conf /root/.kube/config
- until KUBECONFIG=/root/.kube/config kubectl get node; do sleep 2;done
- KUBECONFIG=/root/.kube/config kubectl -n kube-system create secret generic hcloud --from-literal=token={{.HcloudToken}} --from-literal=network={{.HcloudNetwork}}
# Download and install latest hcloud cli release for easier debugging on host
- curl -s https://api.github.com/repos/hetznercloud/cli/releases/latest | grep browser_download_url | grep linux-amd64 | cut -d '"' -f 4 | wget -qi -
- tar xvzf hcloud-linux-amd64.tar.gz && cp hcloud /usr/bin/hcloud && chmod +x /usr/bin/hcloud
