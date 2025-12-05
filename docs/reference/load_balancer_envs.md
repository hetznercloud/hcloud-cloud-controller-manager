# Load Balancer Environment Variables

This page contains all environment variables, which can be specified to configure the Load Balancer controller of HCCM. Most of them are used to set global defaults.

- Enums are depicted in the `Type` column and possible options are separated via the pipe symbol `|`.

| Annotation | Type | Default | Description |
| --- | --- | --- | --- |
| `HCLOUD_LOAD_BALANCERS_DISABLE_IPV6` | `bool` | `false` | Disables the use of IPv6 for the Load Balancer by default. |
| `HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS` | `bool` | `false` | Disables the use of the private network for ingress by default. |
| `HCLOUD_LOAD_BALANCERS_ENABLED` | `bool` | `true` | Controls whether the load balancer controller of HCCM should run. |
| `HCLOUD_LOAD_BALANCERS_LOCATION` | `string` | `-` | Specifies the default location where the Load Balancer will be created in. Mutually exclusive with hcloudLoadBalancersNetworkZone. |
| `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE` | `string` | `-` | Specifies the default network zone where the Load Balancer will be created in. Mutually exclusive with hcloudLoadBalancersLocation. |
| `HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP` | `bool` | `false` | Configures the Load Balancer to use the private IP for Load Balancer server targets by default. |
