# Private Networks

[Private Networks](https://docs.hetzner.cloud/reference/cloud#networks) can be used to communicate between servers over a dedicated network interface using IP addresses not available publicly.

The hcloud-cloud-controller-manager utilizes them for two features:

1. Use the native routing capability of Private Networks in conjunction with your CNI and eliminate the overhead for an overlay network (e.g. VXLAN).
2. Use private IP targets for LoadBalancers

### Network Attachment Verification

When a network is configured, HCCM uses the [metadata service](https://docs.hetzner.cloud/reference/cloud#server-metadata) to verify whether the specified network is attached to the node it’s running on. This behavior can be disabled by setting the environment variable:

```bash
HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK=true
```

### Node Object Updates

The controller automatically adds the server’s private IP address to the corresponding Kubernetes Node object under the InternalIP field.

## Route Controller

The route controller is part of the HCCM and responsible for creating and deleting routes at Private Networks.

It is enabled by default, if a networking is enabled and a Private Network ID or name is provided. It can be disabled by setting `HCLOUD_NETWORK_ROUTES_ENABLED` to `false`.

To utilize this feature you need to use a CNI, which supports using the native routing capability of the infrastructure. As an example, Cilium can be set to use the [`routing-mode: native`](https://docs.cilium.io/en/stable/network/concepts/routing/#native-routing).

### IP Range Considerations

When using the route controller you need to make some considerations on the IP ranges for the cluster and service CIDR. By default, Kubernetes will allocate a `/24` subnet for each node. Depending on the amount of nodes you plan to add to the cluster you need to choose your cluster CIDR accordingly.

#### Example Values

- Private Network IP range: `10.0.0.0/8`
- Subnet for Cloud Servers & Load Balancers: `10.0.0.0/24`
- Cluster CIDR: `10.244.0.0/16`
- Service CIDR: `10.43.0.0/16`
