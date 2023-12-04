# Deployment with Networks support

The deployment of the Hetzner Cloud Cloud Controller Manager with Networks support is quite different to the normal one.
If you would like to use the version without using our Networks feature, you can follow the Steps at "[Basic deployment](../README.md#deployment)".

We assume, that you have knowledge about Kubernetes and the Hetzner Cloud.

## How to deploy
 1. Create a new Network via `hcloud-cli` (`hcloud network create --name my-network --ip-range=10.0.0.0/8`)or the [Hetzner Cloud Console](https://console.hetzner.cloud)
 2. Add each Node of the Cluster to the Hetzner Cloud Network
 3. Download the latest deployment file with networks support from [Github](https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest) to your local machine
 4. Change the `--cluster-cidr=` flag in the deployment file to fit your pod range. Default is `10.244.0.0/16`.
 5. Create a new secret containing a Hetzner Cloud API Token and the name or the ID of the Network you want to use `kubectl -n kube-system create secret generic hcloud --from-literal=token=<hcloud API token> --from-literal=network=<hcloud Network_ID_or_Name>`
 6. Deploy the deployment file `kubectl -n kube-system apply -f path/to/your/deployment.yaml`
 7. (Recommended) Deploy a CNI (like Cilium `kubectl create -f https://raw.githubusercontent.com/cilium/cilium/v1.5/examples/kubernetes/<kubernetes-version>/cilium.yaml` - please replace `<kubernetes-version>` with your version like `1.15`)


When deploying Cilium, make sure that you have set `tunnel: disabled` and `nativeRoutingCIDR` to your clusters subnet CIDR. If you are using Cilium < 1.9.0 you also have to set `blacklist-conflicting-routes: false`.

After this, you should be able to see the correct routes in the [Hetzner Cloud Console](https://console.hetzner.cloud) or via `hcloud-cli` (`hcloud network describe <hcloud Network_ID_or_Name>`).

## Considerations on the IP Ranges

The `cluster-cidr` Range must be **within the Hetzner Cloud Network Range**, but **must not overlap with any created subnets**. By default, Kubernetes assigns a `/24` (254 addresses) per Node. Changing the range later on is possible, but requires some work. You should assign a range that is large enough to fit enough nodes. For example, if you plan to use a cluster with 10 nodes, you need to assign at least a `/20` (16 x `/24`) to the `cluster-cidr` flag.

The `service-cidr` Range can be within the Hetzner Cloud Network Range, as long as it does not overlap with any other Subnets. 

Some example values:

- Hetzner Cloud Network Range: `10.0.0.0/16`
- Subnet for Cloud Servers & Load Balancers: `10.0.1.0/24` (254 Servers & LBs, API maximum is 100 members)
- Subnet for Robot vSwitch: `10.0.2.0/24` (254 Servers, API maximum is 100 members)
- Cluster CIDR: `10.0.16.0/20` (up to 16 Nodes)
  - Kubernetes will assign a `/24` to every node:
    - Node 1: `10.0.16.0/24`, Node 2: `10.0.17.0/24`, ...
- Service CIDR: `10.0.8.0/21` (up to 2046 `ClusterIP` services)

## Common Issues

### FailedToCreateRoute

Error Message:

```
Could not create route xy-xy-xy-xy-xy 10.244.0.0/24 for node xy.example.com after 1s: hcloud/CreateRoute: network route destination overlaps with another subnetwork or network route (invalid_input)
```

Solution:
Make sure the cluster-cidr does not overlap with the Hetzner Cloud Subnet. Check [Considerations on the IP Ranges](#considerations-on-the-ip-ranges) for more information.
