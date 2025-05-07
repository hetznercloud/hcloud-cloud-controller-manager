# Kubernetes Cloud Controller Manager for Hetzner Cloud

[![e2e tests](https://github.com/hetznercloud/hcloud-cloud-controller-manager/actions/workflows/test_e2e.yml/badge.svg)](https://github.com/hetznercloud/hcloud-cloud-controller-manager/actions/workflows/test_e2e.yml)
[![Codecov](https://codecov.io/github/hetznercloud/hcloud-cloud-controller-manager/graph/badge.svg?token=Q7pbOoyVpj)](https://codecov.io/github/hetznercloud/hcloud-cloud-controller-manager/tree/main)

The Hetzner Cloud [cloud-controller-manager](https://kubernetes.io/docs/concepts/architecture/cloud-controller/) integrates your Kubernetes cluster with the Hetzner Cloud & Robot APIs.

## Features

- **Node**:
  - Updates your `Node` objects with information about the server from the Cloud & Robot API.
  - Instance Type, Location, Datacenter, Server ID, IPs.
- **Node Lifecycle**:
  - Cleans up stale `Node` objects when the server is deleted in the API.
- **Routes** (if enabled):
  - Routes traffic to the pods through Hetzner Cloud Networks. Removes one layer of indirection in CNIs that support this.
- **Load Balancer**:
  - Watches Services with `type: LoadBalancer` and creates Hetzner Cloud Load Balancers for them, adds Kubernetes Nodes as targets for the Load Balancer.

Read more about cloud controllers in the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/).

### Node Metadata Example

```yaml
apiVersion: v1
kind: Node
metadata:
  labels:
    node.kubernetes.io/instance-type: cx22
    topology.kubernetes.io/region: fsn1
    topology.kubernetes.io/zone: fsn1-dc8
    instance.hetzner.cloud/provided-by: cloud
  name: node
spec:
  podCIDR: 10.244.0.0/24
  providerID: hcloud://123456 # <-- Hetzner Cloud Server ID
status:
  addresses:
    - address: node
      type: Hostname
    - address: 1.2.3.4 # <-- Hetzner Cloud Server public ipv4
      type: ExternalIP
```

## Deployment

This deployment example uses `kubeadm` to bootstrap an Kubernetes
cluster, with [flannel](https://github.com/coreos/flannel) as overlay
network agent. Feel free to adapt the steps to your preferred method of
installing Kubernetes.

These deployment instructions are designed to guide with the
installation of the `hcloud-cloud-controller-manager` and are by no
means an in depth tutorial of setting up Kubernetes clusters.
**Previous knowledge about the involved components is required.**

Please refer to the [kubeadm cluster creation
guide](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/),
which these instructions are meant to augment and the [kubeadm
documentation](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/).

1. The cloud controller manager adds the labels when a node is added to
   the cluster. For current Kubernetes versions, this means we
   have to add the `--cloud-provider=external` flag to the `kubelet`. How you
   do this depends on your Kubernetes distribution. With `kubeadm` you can
   either set it in the kubeadm config
   ([`nodeRegistration.kubeletExtraArgs`][kubeadm-config]) or through a systemd
   drop-in unit `/etc/systemd/system/kubelet.service.d/20-hcloud.conf`:

   ```ini
   [Service]
   Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
   ```

   Note: the `--cloud-provider` flag is deprecated since K8S 1.19. You
   will see a log message regarding this. For now (v1.32) it is still required.

2. Now the control plane can be initialized:

   ```sh
   sudo kubeadm init --pod-network-cidr=10.244.0.0/16
   ```

3. Configure kubectl to connect to the kube-apiserver:

   ```sh
   mkdir -p $HOME/.kube
   sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
   sudo chown $(id -u):$(id -g) $HOME/.kube/config
   ```

4. Deploy the flannel CNI plugin:

   ```sh
   kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
   ```

5. Patch the flannel deployment to tolerate the `uninitialized` taint:

   ```sh
   kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{"op":"add","path":"/spec/template/spec/tolerations/-","value":{"key":"node.cloudprovider.kubernetes.io/uninitialized","value":"true","effect":"NoSchedule"}}]'
   ```

6. Create a secret containing your Hetzner Cloud API token.

   ```sh
   kubectl -n kube-system create secret generic hcloud --from-literal=token=<hcloud API token>
   ```

7. Deploy `hcloud-cloud-controller-manager`

   **Using Helm (recommended):**

   ```
   helm repo add hcloud https://charts.hetzner.cloud
   helm repo update hcloud
   helm install hccm hcloud/hcloud-cloud-controller-manager -n kube-system
   ```

   See the [Helm chart README](./chart/README.md) for more info.

   **Legacy installation method**:

   ```sh
   kubectl apply -f https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml
   ```

[kubeadm-config]: https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta4/#kubeadm-k8s-io-v1beta4-NodeRegistrationOptions

## Networks support

When you use the Cloud Controller Manager with networks support, the CCM is in favor of allocating the IPs (& setup the
routing) (Docs: https://kubernetes.io/docs/concepts/architecture/cloud-controller/#route-controller). The CNI plugin you
use needs to support this k8s native functionality (Cilium does it, I don't know about Calico & WeaveNet), so basically
you use the Hetzner Cloud Networks as the underlying networking stack.

When you use the CCM without Networks support it just disables the RouteController part, all other parts work completely
the same. Then just the CNI is in charge of making all the networking stack things. Using the CCM with Networks support
has the benefit that your node is connected to a private network so the node doesn't need to encrypt the connections and
you have a bit less operational overhead as you don't need to manage the Network.

If you want to use the Hetzner Cloud `Networks` Feature, head over to
the [Deployment with Networks support
documentation](./docs/deploy_with_networks.md).

If you manage the network yourself it might still be required to let the CCM know about private networks. For example,
even with a self-managed network, it's still possible to enable private network attachment of CCM-provisioned Load
Balancers by setting the `load-balancer.hetzner.cloud/use-private-ip` annotation to `true` on the Kubernetes Service.
This functionality requires setting the following environment variables in the CCM deployment:

```
          env:
            - name: HCLOUD_NETWORK
              valueFrom:
                secretKeyRef:
                  name: hcloud
                  key: network
            - name: HCLOUD_NETWORK_ROUTES_ENABLED
              value: "false"
```

You also need to add the network name/ID to the
secret: `kubectl -n kube-system create secret generic hcloud --from-literal=token=<hcloud API token> --from-literal=network=<hcloud Network_ID_or_Name>`
.

## Kube-proxy mode IPVS and HCloud LoadBalancer

If `kube-proxy` is run in IPVS mode, the `Service` manifest needs to have the
annotation `load-balancer.hetzner.cloud/hostname` where the FQDN resolves to the HCloud LoadBalancer IP.

See https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/212

## Versioning policy

We aim to support the latest three versions of Kubernetes. When a Kubernetes
version is marked as _End Of Life_, we will stop support for it and remove the
version from our CI tests. This does not necessarily mean that the
Cloud Controller Manager does not still work with this version. We will
not fix bugs related only to an unsupported version.

Current Kubernetes Releases: https://kubernetes.io/releases/

### With Networks support

| Kubernetes | Cloud Controller Manager |                                                                                             Deployment File |
| ---------- | -----------------------: | ----------------------------------------------------------------------------------------------------------: |
| 1.32       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.31       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.30       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.29       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.28       |                  v1.20.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.20.0/ccm-networks.yaml |
| 1.27       |                  v1.20.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.20.0/ccm-networks.yaml |
| 1.26       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm-networks.yaml |
| 1.25       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm-networks.yaml |
| 1.24       |                  v1.17.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.17.2/ccm-networks.yaml |
| 1.23       |                  v1.13.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.13.2/ccm-networks.yaml |

### Without Networks support

| Kubernetes | Cloud Controller Manager |                                                                                    Deployment File |
| ---------- | -----------------------: | -------------------------------------------------------------------------------------------------: |
| 1.32       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.31       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.30       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.29       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.28       |                  v1.20.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.20.0/ccm.yaml |
| 1.27       |                  v1.20.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.20.0/ccm.yaml |
| 1.26       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm.yaml |
| 1.25       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm.yaml |
| 1.24       |                  v1.17.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.17.2/ccm.yaml |
| 1.23       |                  v1.13.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.13.2/ccm.yaml |

## Development

### Setup a development environment

To set up a development environment, make sure you installed the following tools:

- [tofu](https://opentofu.org/)
- [k3sup](https://github.com/alexellis/k3sup)
- [docker](https://www.docker.com/)
- [skaffold](https://skaffold.dev/)

1. Configure a `HCLOUD_TOKEN` in your shell session.

> [!WARNING]
> The development environment runs on Hetzner Cloud servers which will induce costs.

2. Deploy the development cluster:

```sh
make -C dev up
```

3. Load the generated configuration to access the development cluster:

```sh
source dev/files/env.sh
```

4. Check that the development cluster is healthy:

```sh
kubectl get nodes -o wide
```

5. Start developing hcloud-cloud-controller-manager in the development cluster:

```sh
skaffold dev
```

On code change, skaffold will rebuild the image, redeploy it and print all logs.

⚠️ Do not forget to clean up the development cluster once are finished:

```sh
make -C dev down
```

### Run the unit tests

To run the unit tests, make sure you installed the following tools:

- [Go](https://go.dev/)

1. Run the following command to run the unit tests:

```sh
go test ./...
```

### Run the kubernetes e2e tests

Before running the e2e tests, make sure you followed the [Setup a development environment](#setup-a-development-environment) steps.

1. Run the kubernetes e2e tests using the following command:

```sh
source dev/files/env.sh
go test ./tests/e2e -tags e2e -v
```

### Development with Robot

If you want to work on the Robot support, you need to make some changes to the above setup.

This requires that you have a Robot Server in the same account you use for the development. The server needs to be setup with the Ansible Playbook `dev/robot/install.yml` and configured in `dev/robot/install.yml`.

1. Set these environment variables:

```shell
export ROBOT_ENABLED=true

export ROBOT_USER=<Your Robot User>
export ROBOT_PASSWORD=<Your Robot Password>
```

2. Continue with the environment setup until you reach the `skaffold` step. Run `skaffold dev --profile=robot` instead.

3. We have another suite of tests for Robot. You can run these with:

```sh
go test ./tests/e2e -tags e2e,robot -v
```

## License

Apache License, Version 2.0
