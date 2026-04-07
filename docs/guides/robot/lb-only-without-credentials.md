# Load Balancer-Only Setup Without Robot API Credentials

If you manage Robot nodes externally (e.g., via Talos or another provisioning tool) and only need the HCCM to add Robot servers as Load Balancer IP targets, you can run without Robot API credentials. This avoids exposing account-wide Robot API credentials to the cluster.

In this mode, the HCCM derives Load Balancer targets from the Kubernetes Node's `InternalIP` instead of querying the Robot API. The Node Controller and Node Lifecycle Controller are not available, as they require the Robot API to fetch server metadata.

## Prerequisites

- Nodes must be initialized with a provider ID
- Robot servers must be connected to a vSwitch with an `InternalIP` configured on each Node.

## Setup

1. Create a secret without Robot credentials:

```bash
export HCLOUD_TOKEN=<your-hcloud-token>
export HCLOUD_NETWORK=<your-network-id>
kubectl -n kube-system create secret generic hcloud \
    --from-literal=token=$HCLOUD_TOKEN \
    --from-literal=network=$HCLOUD_NETWORK
```

2. Install the Helm chart with Robot enabled, node and route controllers disabled, and private IPs configured:

```bash
helm repo add hcloud https://charts.hetzner.cloud
helm repo update hcloud
helm install hcloud/hcloud-cloud-controller-manager \
    --set robot.enabled=true \
    --set networking.enabled=false \
    --set env.HCLOUD_NETWORK_ROUTES_ENABLED.value="false" \
    --set env.HCLOUD_NETWORK.valueFrom.secretKeyRef.name=hcloud \
    --set env.HCLOUD_NETWORK.valueFrom.secretKeyRef.key=network \
    --set args='{--controllers=*\,-cloud-node\,-cloud-node-lifecycle}'
```

3. Verify that your Robot Nodes have a `ProviderID` and an `InternalIP`:

```bash
kubectl get nodes -o 'custom-columns=NAME:.metadata.name,PROVIDER-ID:.spec.providerID,INTERNAL-IP:.status.addresses[?(@.type=="InternalIP")].address'
```

4. Annotate your Services with `load-balancer.hetzner.cloud/use-private-ip: "true"` to use the `InternalIP` as the Load Balancer target. See the [Private Networks guide](./private-networks.md) for more details.
