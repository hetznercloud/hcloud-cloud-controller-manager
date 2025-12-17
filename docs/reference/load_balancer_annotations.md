# Load Balancer Annotations

This page contains all annotations, which can be specified at a Service of type `LoadBalancer` to configure a Hetzner Cloud Load Balancer. Additionally, this includes read-only annotations, which are set by the Cloud Controller Manager.

- Read-only annotations are set by the Cloud Controller Manager.
- Enums are depicted in the `Type` column and possible options are separated via the pipe symbol `|`.

| Name | Type | Default | Read-only | Description |
| --- | --- | --- | --- | --- |
| `load-balancer.hetzner.cloud/ipv4` | `string` | `-` | `Yes` | Is the public IPv4 address assigned to the Load Balancer by the backend. |
| `load-balancer.hetzner.cloud/ipv4-rdns` | `string` | `-` | `Yes` | Is the reverse DNS record assigned to the IPv4 address of the Load Balancer. |
| `load-balancer.hetzner.cloud/ipv6` | `string` | `-` | `Yes` | Is the public IPv6 address assigned to the Load Balancer by the backend. |
| `load-balancer.hetzner.cloud/ipv6-rdns` | `string` | `-` | `Yes` | Is the reverse DNS record assigned to the IPv6 address of the Load Balancer. |
| `load-balancer.hetzner.cloud/ipv6-disabled` | `bool` | `false` | `No` | Disables the use of IPv6 for the Load Balancer. Set this annotation if you use external-dns. |
| `load-balancer.hetzner.cloud/name` | `string` | `-` | `No` | Is the name of the Load Balancer. The name will be visible in the Hetzner Cloud API console. |
| `load-balancer.hetzner.cloud/disable-public-network` | `bool` | `false` | `No` | Disables the public network of the Hetzner Cloud Load Balancer. It will still have a public network assigned, but all traffic is routed over the private network. |
| `load-balancer.hetzner.cloud/disable-private-ingress` | `bool` | `false` | `No` | Disables the use of the private network for ingress. |
| `load-balancer.hetzner.cloud/use-private-ip` | `bool` | `false` | `No` | Configures the Load Balancer to use the private IP for Load Balancer server targets. |
| `load-balancer.hetzner.cloud/private-ipv4` | `string` | `-` | `No` | Specifies the IPv4 address to assign to the load balancer in the private network that it's attached to. |
| `load-balancer.hetzner.cloud/private-subnet-ip-range` | `string` | `-` | `No` | Specifies an existing subnet to which the load balancer will be attached. The value must be in the CIDR notation. The subnet must belong to the network defined in the CCM configuration and must already exist. See: https://docs.hetzner.cloud/reference/cloud#network-actions-add-a-subnet-to-a-network |
| `load-balancer.hetzner.cloud/hostname` | `string` | `-` | `No` | Specifies the hostname of the Load Balancer. This will be used as ingress address instead of the Load Balancer IP addresses if specified. |
| `load-balancer.hetzner.cloud/protocol` | `tcp \| http \| https` | `tcp` | `No` | Specifies the protocol of the service. |
| `load-balancer.hetzner.cloud/algorithm-type` | `round_robin \| least_connections` | `round_robin` | `No` | Specifies the algorithm type of the Load Balancer. |
| `load-balancer.hetzner.cloud/type` | `string` | `lb11` | `No` | Specifies the type of the Load Balancer. |
| `load-balancer.hetzner.cloud/location` | `string` | `-` | `No` | Specifies the location where the Load Balancer will be created in. Changing the location to a different value after the load balancer was created has no effect. In order to move a load balancer to a different location it is necessary to delete and re-create it. Note, that this will lead to the load balancer getting new public IPs assigned. Mutually exclusive with `load-balancer.hetzner.cloud/network-zone`. |
| `load-balancer.hetzner.cloud/network-zone` | `string` | `-` | `No` | Specifies the network zone where the Load Balancer will be created in. Changing the network zone to a different value after the load balancer was created has no effect.  In order to move a load balancer to a different network zone it is necessary to delete and re-create it. Note, that this will lead to the load balancer getting new public IPs assigned. Mutually exclusive with `load-balancer.hetzner.cloud/location`. |
| `load-balancer.hetzner.cloud/node-selector` | `string` | `-` | `No` | Can be set to restrict which Nodes are added as targets to the Load Balancer. It accepts a Kubernetes label selector string, using either the set-based or equality-based formats. If the selector can not be parsed, the targets in the Load Balancer are not updated and an Event is created with the error message. Format: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors |
| `load-balancer.hetzner.cloud/uses-proxyprotocol` | `bool` | `false` | `No` | Specifies if the Load Balancer services should use the proxy protocol. |
| `load-balancer.hetzner.cloud/http-cookie-name` | `string` | `-` | `No` | Specifies the cookie name when using  HTTP or HTTPS as protocol. |
| `load-balancer.hetzner.cloud/http-cookie-lifetime` | `int` | `-` | `No` | Specifies the lifetime of the HTTP cookie. |
| `load-balancer.hetzner.cloud/certificate-type` | `uploaded \| managed` | `uploaded` | `No` | Defines the type of certificate the Load Balancer should use. |
| `load-balancer.hetzner.cloud/http-certificates` | `string` | `-` | `No` | A comma separated list of IDs or Names of Certificates assigned to the service. |
| `load-balancer.hetzner.cloud/http-managed-certificate-name` | `string` | `-` | `No` | Contains the name of the managed certificate to create by the Cloud Controller manager. Ignored if `load-balancer.hetzner.cloud/certificate-type` is missing or set to "uploaded". |
| `load-balancer.hetzner.cloud/http-managed-certificate-domains` | `string` | `-` | `No` | Contains a comma separated list of the domain names of the managed certificate. All domains are used to create a single managed certificate. |
| `load-balancer.hetzner.cloud/http-redirect-http` | `bool` | `false` | `No` | Create a redirect from HTTP to HTTPS. |
| `load-balancer.hetzner.cloud/http-sticky-sessions` | `bool` | `false` | `No` | Enables the sticky sessions feature of Hetzner Cloud HTTP Load Balancers. |
| `load-balancer.hetzner.cloud/health-check-protocol` | `tcp \| http \| https` | `tcp` | `No` | Sets the protocol the health check should be performed over. |
| `load-balancer.hetzner.cloud/health-check-port` | `int` | `-` | `No` | Specifies the port the health check is be performed on. |
| `load-balancer.hetzner.cloud/health-check-interval` | `int` | `-` | `No` | Specifies the interval in which time we perform a health check in seconds. |
| `load-balancer.hetzner.cloud/health-check-timeout` | `int` | `-` | `No` | Specifies the timeout of a single health check. |
| `load-balancer.hetzner.cloud/health-check-retries` | `int` | `-` | `No` | Specifies the number of time a health check is retried until a target is marked as unhealthy. |
| `load-balancer.hetzner.cloud/health-check-http-domain` | `string` | `-` | `No` | Specifies the domain we try to access when performing the health check. |
| `load-balancer.hetzner.cloud/health-check-http-path` | `string` | `-` | `No` | Specifies the path we try to access when performing the health check. |
| `load-balancer.hetzner.cloud/health-check-http-validate-certificate` | `bool` | `-` | `No` | Specifies whether the health check should validate the SSL certificate that comes from the target nodes. |
| `load-balancer.hetzner.cloud/http-status-codes` | `string` | `-` | `No` | Is a comma separated list of HTTP status codes which we expect. |
| `load-balancer.hetzner.cloud/id` | `string` | `-` | `Yes` | Is the ID assigned to the Hetzner Cloud Load Balancer by the backend. Deprecated: This annotation is not used. It is reserved for possible future use. |
