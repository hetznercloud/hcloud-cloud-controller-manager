#cloud-config
write_files:
- content: |
    net.bridge.bridge-nf-call-ip6tables = 1
    net.bridge.bridge-nf-call-iptables = 1
  path: /etc/sysctl.d/k8s.conf
- content: |
    alias k="kubectl"
    alias ksy="kubectl -n kube-system"
    alias kgp="kubectl get pods"
    alias kgs="kubectl get services"
    export HCLOUD_TOKEN={{.HcloudToken}}
  path: /root/.bashrc
runcmd:
- sysctl --system
- apt install -y apt-transport-https curl
- export INSTALL_K3S_VERSION={{.K8sVersion}}
# Download and install latest hcloud cli release for easier debugging on host
- curl -s https://api.github.com/repos/hetznercloud/cli/releases/latest | grep browser_download_url | grep linux-amd64 | cut -d '"' -f 4 | wget -qi -
- tar xvzf hcloud-linux-amd64.tar.gz && cp hcloud /usr/bin/hcloud && chmod +x /usr/bin/hcloud
{{if .IsClusterServer}}
- curl -sfL https://get.k3s.io | sh -s - --disable servicelb --disable traefik --disable-cloud-controller --kubelet-arg="cloud-provider=external" --disable metrics-server {{if not .UseFlannel }}--flannel-backend=none{{ end }}
- mkdir -p /opt/cni/bin
- ln -s /var/lib/rancher/k3s/data/current/bin/loopback /opt/cni/bin/loopback # Workaround for https://github.com/k3s-io/k3s/issues/219
- ln -s /var/lib/rancher/k3s/data/current/bin/bridge /opt/cni/bin/bridge # Workaround for https://github.com/k3s-io/k3s/issues/219
- ln -s /var/lib/rancher/k3s/data/current/bin/host-local /opt/cni/bin/host-local # Workaround for https://github.com/k3s-io/k3s/issues/219
- ln -s /var/lib/rancher/k3s/data/current/bin/portmap /opt/cni/bin/portmap # Workaround for https://github.com/k3s-io/k3s/issues/219
- mkdir -p /root/.kube
- cp -i /etc/rancher/k3s/k3s.yaml /root/.kube/config
- until KUBECONFIG=/root/.kube/config kubectl get node; do sleep 2;done
- KUBECONFIG=/root/.kube/config kubectl -n kube-system create secret generic hcloud --from-literal=token={{.HcloudToken}} --from-literal=network={{.HcloudNetwork}}
{{else}}
- curl -sfL https://get.k3s.io | {{.JoinCMD}} sh -s - --kubelet-arg="cloud-provider=external"
- sleep 10 # to get the joining work
{{end}}
