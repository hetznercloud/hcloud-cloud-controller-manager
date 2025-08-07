# Private Network Setup

This guide teaches you how to setup HCCM with support for Private Networks. Please familiarize yourself with the explanation document about [Private Networks](../explanation/private-networks.md).

Before setting up the hcloud-cloud-controller-manager you need to configure your cluster appropriately. When creating your cluster you need to provide the `kubelet` [option](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/#options) `--cloud-provider=external`. How this is done depends on your Kubernetes distribution.

By default, the HCCM's route controller is enabled. For this reason, you need to choose an appropriate CNI plugin, which supports native routing capabilities of the underlying infrastructure. As an example, Cilium can be set to use the [`routing-mode: native`](https://docs.cilium.io/en/stable/network/concepts/routing/#native-routing).

1. Select the appropriate IP ranges for your cluster. You can reference the [explanation document](../explanation/private-networks.md).

2. Create the Private Network from the [Hetzner Cloud Console](https://console.hetzner.cloud/) or via the [`hcloud-cli`](https://github.com/hetznercloud/cli):

```bash
hcloud network create --name my-network --ip-range=10.0.0.0/8
```

3. Add your nodes to the network.

4. Provision your Kubernetes cluster with the Kubelet option `--cloud-provider=external`.

5. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/) as described in [this document](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/).

6. Create a secret containing your Hetzner Cloud API token and your Private Network ID or name:

```bash
kubectl -n kube-system create secret generic hcloud \
    --from-literal=token=<hcloud API token> \
    --from-literal=network=<hcloud network-id-or-name>
```

7. Add the Helm repository:

```bash
helm repo add hcloud https://charts.hetzner.cloud
helm repo update hcloud
```

8. Install the chart:

```bash
helm install hccm hcloud/hcloud-cloud-controller-manager -n kube-system \
    --set networking.enabled=true \
    --set networking.clusterCIDR=<cluster-cidr>
```

9. Install your CNI plugin.
