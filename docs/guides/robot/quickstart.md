# Quickstart

Before setting up the hcloud-cloud-controller-manager you need to configure your cluster appropriately. When creating your cluster you need to provide the `kubelet` [option](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/#options) `--cloud-provider=external`. How this is done depends on your Kubernetes distribution.

1. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/) as described in [this document](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/).

2. Export your Robot credentials and Hetzner Cloud API token as environment variables:

```bash
export HCLOUD_TOKEN=<your-hcloud-token>
export ROBOT_USER=<your-robot-user-name>
export ROBOT_PASSWORD=<your-robot-password>
kubectl -n kube-system create secret generic hcloud \
    --from-literal=token=$HCLOUD_TOKEN \
    --from-literal=robot-user=$ROBOT_USER \
    --from-literal=robot-password=$ROBOT_PASSWORD
```

3. Install the Helm chart:

```bash
helm repo add hcloud https://charts.hetzner.cloud
helm repo update hcloud
helm install hcloud/hcloud-cloud-controller-manager --set robot.enabled=true
```
