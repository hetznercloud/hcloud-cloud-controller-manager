# Kubernetes Cloud Controller Manager for Hetzner Cloud
[![GitHub Actions status](https://github.com/hetznercloud/hcloud-cloud-controller-manager/workflows/Run%20tests/badge.svg)](https://github.com/hetznercloud/hcloud-cloud-controller-manager/actions)

The Hetzner Cloud cloud controller manager integrates your Kubernets cluster with the Hetzner Cloud API.
Read more about kubernetes cloud controller managers in the [kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/).

## Features

- **instances interface**: adds the server type to the `beta.kubernetes.io/instance-type` label, sets the external ipv4 and ipv6 addresses and deletes nodes from Kubernetes that were deleted from the Hetzner Cloud.
- **zones interface**: makes Kubernetes aware of the failure domain of the server by setting the `failure-domain.beta.kubernetes.io/region` and `failure-domain.beta.kubernetes.io/zone` labels on the node.
- **Private Networks**: allows to use Hetzner Cloud Private Networks for your pods traffic.
- **Load Balancers**: allows to use Hetzner Cloud Load Balancers with Kubernetes Services


## Example

```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    flannel.alpha.coreos.com/backend-data: '{"VtepMAC":"06:b3:ee:88:92:36"}'
    flannel.alpha.coreos.com/backend-type: vxlan
    flannel.alpha.coreos.com/kube-subnet-manager: "true"
    flannel.alpha.coreos.com/public-ip: 78.46.208.178
    node.alpha.kubernetes.io/ttl: "0"
    volumes.kubernetes.io/controller-managed-attach-detach: "true"
  creationTimestamp: 2018-01-24T15:59:45Z
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: cx11 # <-- server type
    beta.kubernetes.io/os: linux
    topology.kubernetes.io/region: fsn1 # <-- location
    topology.kubernetes.io/zone: fsn1-dc8 # <-- datacenter
    kubernetes.io/hostname: master
    node-role.kubernetes.io/master: ""
  name: master
  resourceVersion: "183932"
  selfLink: /api/v1/nodes/master
  uid: 98acdedc-011f-11e8-9ed3-9600000780bf
spec:
  externalID: master
  podCIDR: 10.244.0.0/24
  providerID: hcloud://123456 # <-- Server ID
status:
  addresses:
  - address: master
    type: Hostname
  - address: 78.46.208.178 # <-- public ipv4
    type: ExternalIP
```

## Deployment

This deployment example uses `kubeadm` to bootstrap an Kubernetes cluster, with [flannel](https://github.com/coreos/flannel) as overlay network agent. Feel free to adapt the steps to your preferred method of installing Kubernetes.

These deployment instructions are designed to guide with the installation of the `hcloud-cloud-controller-manager` and are by no means an in depth tutorial of setting up Kubernetes clusters.
**Previous knowledge about the involved components is required.**

Please refer to the [kubeadm cluster creation guide](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/), which these instructions are meant to argument and the [kubeadm documentation](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/).

1. The cloud controller manager adds its labels when a node is added to the cluster. This means we have to add the `--cloud-provider=external` flag to the `kubelet` before initializing the cluster master with `kubeadm init`.
To do accomplish this we add this systemd drop-in unit:
`/etc/systemd/system/kubelet.service.d/20-hcloud.conf`

```
[Service]
Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
```

2. Now the cluster master can be initialized:

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

7. Deploy the `hcloud-cloud-controller-manager`:

```
kubectl apply -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/ccm.yaml

```

If you want to use the Hetzner Cloud `Networks` Feature, head over to the [Deployment with Networks support documentation](./docs/deploy_with_networks.md).


## E2E Tests

The Hetzner Cloud cloud controller manager was tested against all supported Kubernetes versions. You can run the tests with the following commands. Keep in mind, that these tests run on real cloud servers and will create Load Balancers that will be billed. 

**Test Server Setup:** 
1x CPX21 (Ubuntu 18.04)

**Requirements: Docker and Go 1.15**

1. Configure your environment correctly

```bash
export HCLOUD_TOKEN=<specifiy a project token>
export K8S_VERSION=1.19.3 # The specific (latest) version is needed here
export USE_SSH_KEYS=key1,key2 # Name or IDs of your SSH Keys within the Hetzner Cloud, the servers will be accessable with that keys
export USE_NETWORKS=yes # if `yes` this identidicates that the tests should provision the server with cilium as CNI and also enable the Network related tests
## Optional configuration env vars:
export TEST_DEBUG_MODE=yes # With this env you can toggle the output of the provision and test commands. With `yes` it will log the whole output to stdout
```

2. Run the tests

```bash
go test $(go list ./... | grep e2etests) -v -timeout 60m
```
The tests will now run and cleanup themselves afterwards. Sometimes it might happen that you need to clean up the project manually via the [Hetzner Cloud Console](https://console.hetzner.cloud) or the [hcloud-cli](https://github.com/hetznercloud/cli) .

For easier debugging on the server we always configure the latest version of the [hcloud-cli](https://github.com/hetznercloud/cli) with the given `HCLOUD_TOKEN` and a few bash aliases on the host:

```bash
alias k="kubectl"
alias ksy="kubectl -n kube-system"
alias kgp="kubectl get pods"
alias kgs="kubectl get services"
```

## License

Apache License, Version 2.0
