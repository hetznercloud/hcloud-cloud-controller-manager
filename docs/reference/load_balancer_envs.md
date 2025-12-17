# Load Balancer Environment Variables

This page contains all environment variables, which can be specified to configure the Load Balancer controller of hcloud-cloud-controller-manager.

Some environment variables define global defaults. These defaults can be overridden by setting the corresponding annotation. If you remove such an annotation while a global default is configured, the global default will be applied again.

Enums are depicted in the `Type` column and possible options are separated via the pipe symbol `|`.

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| `HCLOUD_LOAD_BALANCERS_ENABLED` | `bool` | `true` | Controls whether the load balancer controller of HCCM should run. |
| `HCLOUD_LOAD_BALANCERS_LOCATION` | `string` | `-` | Specifies the default location where the Load Balancer will be created in. Mutually exclusive with `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE`. |
| `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE` | `string` | `-` | Specifies the default network zone where the Load Balancer will be created in. Mutually exclusive with `HCLOUD_LOAD_BALANCERS_LOCATION`. |
| `HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS` | `bool` | `false` | Disables the use of the private network for ingress by default. |
| `HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP` | `bool` | `false` | Configures the Load Balancer to use the private IP for Load Balancer server targets by default. |
| `HCLOUD_LOAD_BALANCERS_DISABLE_IPV6` | `bool` | `false` | Disables the use of IPv6 for the Load Balancer by default. |
| `HCLOUD_LOAD_BALANCERS_ALGORITHM_TYPE` | `round_robin \| least_connections` | `round_robin` | Configures the default Load Balancer algorithm. |
| `HCLOUD_LOAD_BALANCERS_DISABLE_PUBLIC_NETWORK` | `bool` | `false` | Disables the public interface of the Load Balancer by default. |
| `HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_INTERVAL` | `int` | `10` | Configures the default time interval in seconds in which health checks are performed. |
| `HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_RETRIES` | `int` | `3` | Configures the default amount of unsuccessful retries needed until a target is considered unhealthy. |
| `HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_TIMEOUT` | `int` | `15` | Configures the default time in seconds after an attempt is considered a timeout. |
| `HCLOUD_LOAD_BALANCERS_PRIVATE_SUBNET_IP_RANGE` | `string` | `-` | Configures the default IP range in CIDR block notation of the subnet to attach to. |
| `HCLOUD_LOAD_BALANCERS_TYPE` | `string` | `lb11` | Configures the default Load Balancer type this Load Balancer should be created with. |
| `HCLOUD_LOAD_BALANCERS_USES_PROXYPROTOCOL` | `bool` | `false` | Enables the proxyprotocol for a Load Balancer service by default. |
