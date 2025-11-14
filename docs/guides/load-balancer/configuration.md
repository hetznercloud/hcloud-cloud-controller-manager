# Configuration

Load Balancers are configured via Kubernetes [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/). You can find a full list of annotations and their description in [this table](../../reference/load_balancer_annotations.md).

## Cluster-Wide Defaults

For convenience, you can set the following environment variables as cluster-wide defaults, so you don't have to set them on each load balancer service. If a load balancer service has the corresponding annotation set, it overrides the default.

- `HCLOUD_LOAD_BALANCERS_ALGORITHM_TYPE`
- `HCLOUD_LOAD_BALANCERS_DISABLE_IPV6`
- `HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS`
- `HCLOUD_LOAD_BALANCERS_DISABLE_PUBLIC_NETWORK`
- `HCLOUD_LOAD_BALANCERS_ENABLED`
- `HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_INTERVAL`
- `HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_RETRIES`
- `HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_TIMEOUT`
- `HCLOUD_LOAD_BALANCERS_LOCATION` (mutually exclusive with `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE`)
- `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE` (mutually exclusive with `HCLOUD_LOAD_BALANCERS_LOCATION`)
- `HCLOUD_LOAD_BALANCERS_PRIVATE_SUBNET_IP_RANGE`
- `HCLOUD_LOAD_BALANCERS_TYPE`
- `HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP`
- `HCLOUD_LOAD_BALANCERS_USES_PROXYPROTOCOL`
