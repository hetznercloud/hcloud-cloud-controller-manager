# Kubernetes Cloud Controller Manager for Hetzner Cloud & Hetzner Dedicated

[![GitHub Actions status](https://github.com/syself/hetzner-cloud-controller-manager/workflows/Run%20tests/badge.svg)](https://github.com/syself/hetzner-cloud-controller-manager/actions)

The Hetzner Cloud controller manager seamlessly integrates your Kubernetes cluster with both the Hetzner Cloud API and the Robot API.

> This specific fork of the CCM has been enhanced to support Hetzner Dedicated servers and is actively maintained by [Syself](https://syself.com). Its primary purpose is to facilitate the operation of the [Cluster API Provider Integration Hetzner](https://github.com/syself/cluster-api-provider-hetzner).
> If you have inquiries or are contemplating deploying production-grade Kubernetes clusters on Hetzner, we welcome you to reach out to us at [info@syself.com](mailto:info@syself.com?subject=cluster-api-provider-hetzner).

## About the Fork

In the long run, we (Syself) would like to switch to the [upstream ccm](https://github.com/syself/hetzner-cloud-controller-manager/) again.

A lot of changes were made in the upstream fork, and we don't plan to merge them into our fork.

Instead we plan to create PRs in upstream, so that our fork is no longer needed.

Features/PRs which are different in our fork. We should create PRs in upstream for these:

* [separate user agent from HCCM](https://github.com/syself/hetzner-cloud-controller-manager/pull/42): Make this configurable in upstream.
* [ROBOT_DEBUG, show stacktrace on api-calls](https://github.com/syself/hetzner-cloud-controller-manager/pull/41>) Via ROBOT_DEBUG show every stack-trace which uses the robot-API (to debug why rate-limiting was reached)
* [PR add version information to the controller binary](https://github.com/syself/hetzner-cloud-controller-manager/pull/28): Build process is different in upstream.
* [PR Add Github WF for releasing](https://github.com/syself/hetzner-cloud-controller-manager/pull/29)
* [PR entrypoint in container image](https://github.com/syself/hetzner-cloud-controller-manager/pull/25)
* [PR rate limiting hcloud](https://github.com/syself/hetzner-cloud-controller-manager/pull/20)
* Upstream has three `ServerGetList` implementations: cache, rate-limited, mock. Our fork has two: mock, cached.

Additional PRs we should create in upstream, so that we can use upstream instead our fork:

* Make ProviderID configurable (hrobot://NNN vs hcloud://bm-NNN)
* Sort Go imports
* Compare linters of upstream with the linters of our other repos.

PRs which are **not** needed in upstream, because upstream has this feature:

* [PR getInstanceTypeOfRobotServer: convert invalid characters to dashes](https://github.com/syself/hetzner-cloud-controller-manager/pull/40)
* [Make robot client optional for lb client](https://github.com/syself/hetzner-cloud-controller-manager/pull/37): upstream uses ROBOT_ENABLED. We need to set that env var.
* [Fix InstanceExists for baremetal servers, check node name](https://github.com/syself/hetzner-cloud-controller-manager/pull/32)
* [Handle errors of not enough lb targets](https://github.com/syself/hetzner-cloud-controller-manager/pull/22): Max LB targets reached.
* [Fix lb default for disable IPv6](https://github.com/syself/hetzner-cloud-controller-manager/pull/21/files)
* [robot cache](https://github.com/syself/hetzner-cloud-controller-manager/pull/19)
* [Support for robot](https://github.com/syself/hetzner-cloud-controller-manager/pull/1)

If you update the Syself fork, please create two PRs for version updates and code updates. Mixing both in one PR makes things harder to understand.

Files moved by upstream in their fork:

* internal/robot/client/cache/client.go (from Janis Okt 2023) --> internal/robot/cache.go (by Julian Nov 2023)
* internal/robot/client/interface.go --> internal/robot/interface.go
* internal/util/util.go GetEnvDuration() --> internal/config/config.go
* hcloud/util.go --> hcloud/instances_util.go

How to keep our fork up to date: Check which changes were done in upstream. Pick indivual features, if they make sense for us. If unsure,
don't pick a feature. Instead try to get our features into upstream, and update or dependencies.

TODO:

* from quay.io to ghcr.io

## Installing Syself CCM

```sh
helm repo add syself https://charts.syself.com
helm repo update syself

helm upgrade --install ccm syself/ccm-hetzner --version X.Y.Z \
              --namespace kube-system \
              --set privateNetwork.enabled=false
```

See [CAPH docs](https://syself.com/docs/caph/topics/baremetal/creating-workload-cluster#deploying-the-hetzner-cloud-controller-manager) for more details.

---

End of "About the fork"

Docs below that line are likely out of date.

---

## Features

* **instances interface**: adds the server type to the `node.kubernetes.io/instance-type` label, sets the external ipv4 and ipv6 addresses and deletes nodes from Kubernetes that were deleted from the Hetzner Cloud.
* **zones interface**: makes Kubernetes aware of the failure domain of the server by setting the `topology.kubernetes.io/region` and `topology.kubernetes.io/zone` labels on the node.
* **Private Networks**: allows to use Hetzner Cloud Private Networks for your pods traffic.
* **Load Balancers**: allows to use Hetzner Cloud Load Balancers with Kubernetes Services
* **Hetzner Dedicated**: use Baremetal Server and Cloud Servers together

Read more about cloud controllers in the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/).

## Example

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
installation of the `hetzner-cloud-controller-manager` and are by no
means an in depth tutorial of setting up Kubernetes clusters.
**Previous knowledge about the involved components is required.**

Please refer to the [kubeadm cluster creation
guide](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/),
which these instructions are meant to augment and the [kubeadm
documentation](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/).

1. The cloud controller manager adds its labels when a node is added to
   the cluster. For current Kubernetes versions, this means we
   have to add the `--cloud-provider=external` flag to the `kubelet`
   before initializing the control plane with `kubeadm init`. To do
   accomplish this we add this systemd drop-in unit
   `/etc/systemd/system/kubelet.service.d/20-hcloud.conf`:

   ```
   [Service]
   Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
   ```

   Note: the `--cloud-provider` flag is deprecated since K8S 1.19. You
   will see a log message regarding this. For now (v1.26) it is still required.

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

7. Deploy the `hetzner-cloud-controller-manager`:

    **Using Helm (recommended):**

    ```
    helm repo add hcloud https://charts.hetzner.cloud
    helm repo update hcloud
    helm install hccm hcloud/hcloud-cloud-controller-manager -n kube-system
    ```

    See the [Helm chart README](./chart/README.md) for more info.

    **Legacy installation method**:

    ```sh
    kubectl apply -f  https://github.com/syself/hetzner-cloud-controller-manager/releases/latest/download/ccm.yaml
    ```

## Networks support

When you use the Cloud Controller Manager with networks support, the CCM is in favor of allocating the IPs (& setup the
routing) (Docs: <https://kubernetes.io/docs/concepts/architecture/cloud-controller/#route-controller>). The CNI plugin you
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

See <https://github.com/syself/hetzner-cloud-controller-manager/issues/212>

## Versioning policy

We aim to support the latest three versions of Kubernetes. After a new
Kubernetes version has been released we will stop supporting the oldest
previously supported version. This does not necessarily mean that the
Cloud Controller Manager does not still work with this version. However,
it means that we do not test that version anymore. Additionally, we will
not fix bugs related only to an unsupported version.

### With Networks support

| Kubernetes | Cloud Controller Manager |                                                                                             Deployment File |
|------------|-------------------------:|------------------------------------------------------------------------------------------------------------:|
| 1.28       |                     main |  <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml> |
| 1.27       |                     main |  <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml> |
| 1.26       |                     main |  <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml> |
| 1.25       |                     main |  <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml> |
| 1.24       |                  v1.17.2 | <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.17.2/ccm-networks.yaml> |
| 1.23       |                  v1.13.2 | <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.13.2/ccm-networks.yaml> |

### Without Networks support

| Kubernetes | Cloud Controller Manager |                                                                                    Deployment File |
|------------|-------------------------:|---------------------------------------------------------------------------------------------------:|
| 1.28       |                     main |  <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml> |
| 1.27       |                     main |  <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml> |
| 1.26       |                     main |  <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml> |
| 1.25       |                     main |  <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml> |
| 1.24       |                  v1.17.2 | <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.17.2/ccm.yaml> |
| 1.23       |                  v1.13.2 | <https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.13.2/ccm.yaml> |

## Unit tests

To run unit tests locally, execute

```sh
go test $(go list ./... | grep -v e2e) -v
```

Check that your go version is up to date, tests might fail if it is not.

If in doubt, check which go version the `test:unit` section in `.gitlab-ci.yml`
has set in the `image: golang:$VERSION`.

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
go test $(go list ./... | grep e2e) -v -timeout 60m
```

The tests will now run and cleanup themselves afterwards. Sometimes it might happen that you need to clean up the
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

* The kubeconfig will be created under `/tmp/kubeconfig`
* Kubernetes version can be configured via `--k3s-channel`

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

* `docker login` required
* Skaffold is using your own Docker Hub repo to push the HCCM image.
* After the first run, you might need to set the image to "public" on hub.docker.com

On code change, Skaffold will repack the image & deploy it to your test cluster again. It will also stream logs from the hccm Deployment.

*After setting this up, only the command from step 7 is required!*=

### Bare-Metal Guide (Talos)

Alltough this guide is specifically for [TalosOS](https://talos.dev), it should be easily adaptable to any k8s distribution.

0. Setup Hetzner HCloud and Robot API Access

In order for the provider integration hetzner to communicate with the Hetzner API ([HCloud API](https://docs.hetzner.cloud/) + [Robot API](https://robot.your-server.de/doc/webservice/en.html#preface)), we need to create a secret with the access data. The secret must be in the same namespace as the other CRs.

```shell
export HCLOUD_TOKEN="<YOUR-TOKEN>" \
export HETZNER_ROBOT_USER="<YOUR-ROBOT-USER>" \
export HETZNER_ROBOT_PASSWORD="<YOUR-ROBOT-PASSWORD>" \
export HETZNER_SSH_PUB_PATH="<YOUR-SSH-PUBLIC-PATH>" \
export HETZNER_SSH_PRIV_PATH="<YOUR-SSH-PRIVATE-PATH>" \
```

* HCLOUD_TOKEN: The project where your cluster will be placed to. You have to get a token from your HCloud Project.
* HETZNER_ROBOT_USER: The User you have defined in robot under settings / Web
* HETZNER_ROBOT_PASSWORD: The Robot Password you have set in robot under settings/web.
* HETZNER_SSH_PUB_PATH: The Path to your generated Public SSH Key.
* HETZNER_SSH_PRIV_PATH: The Path to your generated Private SSH Key. This is needed because CAPH uses this key to provision the node in Hetzner Dedicated.

1. Make sure to name your root servers on Hetzner Robot with a `bm-` prefix, e.g. `bm-worker-1`
2. Configure worker nodes to use the same name as hostname / node name

worker.yaml

```yaml
machine:
  network:
    hostname: bm-worker-1
```

3. Enable External Cloud Provider

worker.yaml

```yaml
externalCloudProvider:
  enabled: true # Enable external cloud provider.
  # A list of urls that point to additional manifests for an external cloud provider.
  manifests:
    - https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-bare-metal.yaml
```

5. Apply CCM Secrets

```shell
kubectl -n kube-system create secret generic hetzner --from-literal=hcloud=$HCLOUD_TOKEN --from-literal=robot-user=$HETZNER_ROBOT_USER --from-literal=robot-password=$HETZNER_ROBOT_PASSWORD

kubectl -n kube-system create secret generic robot-ssh --from-literal=sshkey-name=cluster --from-file=ssh-privatekey=$HETZNER_SSH_PRIV_PATH --from-file=ssh-publickey=$HETZNER_SSH_PUB_PATH

# Patch the created secret so it is automatically moved to the target cluster later.
kubectl -n kube-system patch secret hetzner -p '{"metadata":{"labels":{"clusterctl.cluster.x-k8s.io/move":""}}}'
```

6. Check if CCM was configured successfully

Get pod name:

```shell
kubectl -n kube-system get pods | grep ccm
```

Example output:

```shell
ccm-ccm-hetzner-86d4f578bb-hmzvm                1/1     Running   0             49m
```

Check logs:

```shell
kubectl -n kube-system logs pods/ccm-ccm-hetzner-86d4f578bb-hmzvm
```

You should see outputs like:

```shell
I1006 08:35:13.996304       1 event.go:294] "Event occurred" object="bm-worker-1" fieldPath="" kind="Node" apiVersion="v1" type="Normal" reason="Synced" message="Node synced successfully"
I1006 08:35:14.554423       1 node_controller.go:465] Successfully initialized node bm-worker-3 with cloud provider
```

## License

Apache License, Version 2.0
