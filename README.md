# Kubernetes Cloud Controller Manager for Hetzner Cloud

[![GitHub Actions status](https://github.com/hetznercloud/hcloud-cloud-controller-manager/workflows/Run%20tests/badge.svg)](https://github.com/hetznercloud/hcloud-cloud-controller-manager/actions)
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
    node.kubernetes.io/instance-type: cx11
    topology.kubernetes.io/region: fsn1
    topology.kubernetes.io/zone: fsn1-dc8
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
   will see a log message regarding this. For now (v1.29) it is still required.

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
    kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/v0.9.1/Documentation/kube-flannel.yml
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

If you manage the network yourself it might still be required to let the CCM know about private networks. You can do
this by adding the environment variable
with the network name/ID in the CCM deployment.

```
          env:
            - name: HCLOUD_NETWORK
              valueFrom:
                secretKeyRef:
                  name: hcloud
                  key: network
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
|------------|-------------------------:|------------------------------------------------------------------------------------------------------------:|
| 1.29       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.28       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.27       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.26       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.25       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm-networks.yaml |
| 1.24       |                  v1.17.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.17.2/ccm-networks.yaml |
| 1.23       |                  v1.13.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.13.2/ccm-networks.yaml |

### Without Networks support

| Kubernetes | Cloud Controller Manager |                                                                                    Deployment File |
|------------|-------------------------:|---------------------------------------------------------------------------------------------------:|
| 1.29       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.28       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.27       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.26       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.25       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm.yaml |
| 1.24       |                  v1.17.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.17.2/ccm.yaml |
| 1.23       |                  v1.13.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.13.2/ccm.yaml |

## Unit tests

To run unit tests locally, execute

```sh
go test ./...
```

Check that your go version is up-to-date, tests might fail if it is not.

If in doubt, check which go version is installed in the [ci.yaml](.github/workflows/ci.yaml) GitHub Actions Workflow:

```yaml
go-version: "1.21"
```

## E2E Tests

The Hetzner Cloud cloud controller manager was tested against all
supported Kubernetes versions. We also test against the same k3s
releases (Sample: When we support testing against Kubernetes 1.20.x we
also try to support k3s 1.20.x). We try to keep compatibility with k3s
but never guarantee this.

You can run the tests with the following commands. Keep in mind, that
these tests run on real cloud servers and will create Load Balancers
that will be billed.

**Test Server Setup:**

1x CPX21 (Ubuntu 18.04)

**Requirements: Docker and Go 1.21**

1. Configure your environment correctly

```bash
export HCLOUD_TOKEN=<specifiy a project token>
export K8S_VERSION=k8s-1.21.0 # The specific (latest) version is needed here
export USE_SSH_KEYS=key1,key2 # Name or IDs of your SSH Keys within the Hetzner Cloud, the servers will be accessable with that keys
export USE_NETWORKS=yes # if `yes` this identidicates that the tests should provision the server with cilium as CNI and also enable the Network related tests
## Optional configuration env vars:
export TEST_DEBUG_MODE=yes # With this env you can toggle the output of the provision and test commands. With `yes` it will log the whole output to stdout
export KEEP_SERVER_ON_FAILURE=yes # Keep the test server after a test failure.
```

2. Run the tests

```bash
go test ./tests/e2e -tags e2e -v -timeout 60m
```

The tests will now run and cleanup themselves afterward. Sometimes it might happen that you need to clean up the
project manually via the [Hetzner Cloud Console](https://console.hetzner.cloud) or
the [hcloud-cli](https://github.com/hetznercloud/cli) .

For easier debugging on the server we always configure the latest version of
the [hcloud-cli](https://github.com/hetznercloud/cli) with the given `HCLOUD_TOKEN` and a few bash aliases on the host:

```bash
alias k="kubectl"
alias ksy="kubectl -n kube-system"
alias kgp="kubectl get pods"
alias kgs="kubectl get services"
```

The test suite is split in three parts:

- **General Part**: Sets up the test env & checks if the HCCM Pod is properly running
   - Build Tag: `e2e`
- **Cloud Part**: Tests regular functionality against a Cloud-only environment
   - Build Tag: `e2e && !robot`
- **Robot Part**: Tests Robot functionality against a Cloud+Robot environment
   - Build Tag: `e2e && robot`

## Local test setup
This repository provides [skaffold](https://skaffold.dev/) to easily deploy / debug this controller on demand

### Requirements
1. Install [hcloud-cli](https://github.com/hetznercloud/cli)
2. Install [k3sup](https://github.com/alexellis/k3sup)
3. Install [cilium](https://github.com/cilium/cilium-cli)
4. Install [docker](https://www.docker.com/)

You will also need to set a `HCLOUD_TOKEN` in your shell session
### Manual Installation guide
1. Create an SSH key

Assuming you already have created an ssh key via `ssh-keygen`
```
hcloud ssh-key create --name ssh-key-ccm-test --public-key-from-file ~/.ssh/id_rsa.pub 
```

2. Create a server
```
hcloud server create --name ccm-test-server --image ubuntu-20.04 --ssh-key ssh-key-ccm-test --type cx11 
```

3. Setup k3s on this server
```
k3sup install --ip $(hcloud server ip ccm-test-server) --local-path=/tmp/kubeconfig --cluster --k3s-channel=v1.23 --k3s-extra-args='--no-flannel --no-deploy=servicelb --no-deploy=traefik --disable-cloud-controller --disable-network-policy --kubelet-arg=cloud-provider=external'
```
- The kubeconfig will be created under `/tmp/kubeconfig`
- Kubernetes version can be configured via `--k3s-channel`

4. Switch your kubeconfig to the test cluster. Very important: exporting this like 
```
export KUBECONFIG=/tmp/kubeconfig
```

5. Install cilium + test your cluster
```
cilium install
```

6. Add your secret to the cluster
```
kubectl -n kube-system create secret generic hcloud --from-literal="token=$HCLOUD_TOKEN"
```

7. Deploy the hcloud-cloud-controller-manager
```
SKAFFOLD_DEFAULT_REPO=your_docker_hub_username skaffold dev
```

- `docker login` required
- Skaffold is using your own Docker Hub repo to push the HCCM image.
- After the first run, you might need to set the image to "public" on hub.docker.com

On code change, Skaffold will repack the image & deploy it to your test cluster again. It will also stream logs from the hccm Deployment.

*After setting this up, only the command from step 7 is required!*=

## License

Apache License, Version 2.0
