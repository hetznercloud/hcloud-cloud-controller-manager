# Deployment with Networks support

The deployment of the Hetzner Cloud Cloud Controller Manager with Networks support is quite different to the normal one.
If you would like to use the version without using our Networks feature, you can follow the Steps at "[Basic deployment](../README.md#deployment)".

We assume, that you have knowledge about Kubernetes and the Hetzner Cloud.

## How to deploy
 1. Create a new Network via `hcloud-cli` (`hcloud network create --name my-network --ip-range=10.0.0.0/8`)or the [Hetzner Cloud Console](https://console.hetzner.cloud)
 2. Add each Node of the Cluster to the Hetzner Cloud Network
 3. Download the latest deployment file with networks support from [Github](https://github.com/syself/hetzner-cloud-controller-manager/releases/latest) to your local machine
 4. Change the `--cluster-cidr=` flag in the deployment file to fit your pod range. Default is `10.244.0.0/16`.
 5. Create a new secret containing a Hetzner Cloud API Token and the name or the ID of the Network you want to use `kubectl -n kube-system create secret generic hcloud --from-literal=token=<hcloud API token> --from-literal=network=<hcloud Network_ID_or_Name>`
 6. Deploy the deployment file `kubectl -n kube-system apply -f path/to/your/deployment.yaml`
 7. (Recommended) Deploy a CNI (like Cilium `kubectl create -f https://raw.githubusercontent.com/cilium/cilium/v1.5/examples/kubernetes/<kubernetes-version>/cilium.yaml` - please replace `<kubernetes-version>` with your version like `1.15`)


When deploying Cilium, make sure that you have set `tunnel: disabled` and `nativeRoutingCIDR` to your clusters subnet CIDR. If you are using Cilium < 1.9.0 you also have to set `blacklist-conflicting-routes: false`.

After this, you should be able to see the correct routes in the [Hetzner Cloud Console](https://console.hetzner.cloud) or via `hcloud-cli` (`hcloud network describe <hcloud Network_ID_or_Name>`).

## Common Issues
#### FailedToCreateRoute
Error Message:
```
Could not create route xy-xy-xy-xy-xy 10.244.0.0/24 for node xy.example.com after 1s: hcloud/CreateRoute: network route destination overlaps with another subnetwork or network route (invalid_input)
```
Solution:
Make sure the cluster-cidr does not overlap with the Hetzner Cloud Network.
For example your Subnetwork could be `10.10.10.0/24` when the *cluster-cidr* is set to `10.244.0.0/16`.
