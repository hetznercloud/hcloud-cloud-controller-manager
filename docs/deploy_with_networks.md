# Deployment with Networks support

The deployment of the Hetzner Cloud Cloud Controller Manager with Networks support is quite different to the normal one. If you would like to use the version without using our Networks feature, you can follow the Steps at "[Basic deployment](../README.md#deployment)".

We assume, that you have knowledge about Kubernetes and the Hetzner Cloud.

## How to deploy
 1. Create a new Network via `hcloud-cli` (`hcloud network create --name my-network --ip-range=10.0.0.0/8`)or the [Hetzner Cloud Console](https://console.hetzner.cloud)
 2. Download the latest deployment file with networks support from [Github](https://github.com/hetznercloud/hcloud-cloud-controller-manager/tree/master/deploy) to your local machine
 3. Change the `--cluster-cidr=` flag in the deployment file to fit your pod range. Default is `10.244.0.0/16`.
 4. Create a new secret containing a Hetzner Cloud API Token and the name or the ID of the Network you want to use `kubectl -n kube-system create secret generic hcloud --from-literal=token=<hcloud API token> --from-literal=network=<hcloud Network_ID_or_Name>`
 5. Deploy the deployment file `kubectl -n kube-system apply -f path/to/your/deployment.yaml`
 6. (Recommended) Deploy a CNI (like Cilium `kubectl create -f https://raw.githubusercontent.com/cilium/cilium/v1.5/examples/kubernetes/<kubernetes-version>/cilium.yaml` - please replace `<kubernetes-version>` with your version like `1.15`)
 
 
When deploying Cilium, make sure that you have set `tunnel: disabled` and `nativeRoutingCIDR` to your clusters subnet CIDR. If you are using Cilium <1.9.0 you also have to set `blacklist-conflicting-routes: false`. 
 
After this, you should be able to see the correct routes in the [Hetzner Cloud Console](https://console.hetzner.cloud) or via `hcloud-cli` (`hcloud networks describe <hcloud Network_ID_or_Name>`).
