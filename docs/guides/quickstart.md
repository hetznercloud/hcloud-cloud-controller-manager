# Quick start

Before setting up the hcloud-cloud-controller-manager you need to configure your cluster appropriately. When creating your cluster you need to provide the `kubelet` [option](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/#options) `--cloud-provider=external`. How this is done depends on your Kubernetes distribution.

1. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/) as described in [this document](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/).

2. Create a secret containing your Hetzner Cloud API token:

```bash
kubectl -n kube-system create secret generic hcloud --from-literal=token=<hcloud API token>
```

3. Add the Helm repository:

```bash
helm repo add hcloud https://charts.hetzner.cloud
helm repo update hcloud
```

4. Install the chart:

```bash
helm install hccm hcloud/hcloud-cloud-controller-manager -n kube-system
```
