# Configuration

Load Balancers are configured via Kubernetes [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/). You can find a full list of annotations and their description in [this table](TODO).

## Cluster-Wide Defaults

For convenience, you can set the following environment variables as cluster-wide defaults, so you don't have to set them on each load balancer service. If a load balancer service has the corresponding annotation set, it overrides the default.

- `HCLOUD_LOAD_BALANCERS_LOCATION` (mutually exclusive with `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE`)
- `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE` (mutually exclusive with `HCLOUD_LOAD_BALANCERS_LOCATION`)
- `HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS`
- `HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP`
- `HCLOUD_LOAD_BALANCERS_ENABLED`
