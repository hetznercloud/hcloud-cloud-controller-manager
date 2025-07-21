# Controllers

The hcloud-cloud-controller-manager consists of multiple controllers, which work independently of each other. All controllers are defined in the [`cloudprovider`](https://pkg.go.dev/k8s.io/cloud-provider) library according to the [cloud controller manager architecture](https://kubernetes.io/docs/concepts/architecture/cloud-controller/).

The service and route controller are not required and can be disabled independently. Especially the routes controller requires configuring your CNI accordingly.

## Node Controller

The node controller manages the lifecycle of nodes by tracking their status. It also provides the cluster with metadata about each node, such as the provider ID.

When the kubelet flag `--cloud-provider=external` is set, the node is automatically tainted with `node.cloudprovider.kubernetes.io/uninitialized`. This prevents workloads from being scheduled on the node until it has been fully initialized by the cloud controller manager, which supplies cloud-specific information via the Hetzner Cloud API. Additionally, this flag tells Kubernetes to ignore local metadata and instead use the Hetzner Cloud API to obtain node details.

## Service Controller

The service controller is responsible for creating and updating Load Balancers, which are created through Kubernetes service objects of type `LoadBalancer`. Additionally, it handles configuration of Load Balancers via Kubernetes annotations at the Service object.

The service controller can be disabled by setting the environment variable:

```bash
HCLOUD_LOAD_BALANCERS_ENABLED="false" # (Default: true)
```

## Route Controller

When using Private Networks in your Kubernetes cluster the route controller is responsible for creating native routes in the Private Network, to allow Pods to communicate directly without the need for an overlay network. To learn more about how private networks can be integrated with the HCCM you can reference the [explanation document](private-networks.md).

The route controller can be disabled by setting the environment variable:

```bash
HCLOUD_NETWORK_ROUTES_ENABLED="false" # (Default: true)
```
