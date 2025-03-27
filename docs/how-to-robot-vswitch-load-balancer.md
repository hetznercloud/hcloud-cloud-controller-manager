# How to attach load balancers to Robot private IPs

With our HCCM release v1.24.0 we introduced the option to configure InternalIPs for Robot servers. This allows creating a cluster with private networks and a mixture of Robot and Cloud servers. Using the routing feature of private networks is not supported, so it requires a CNI plugin with encapsulation methods, such as Cilium with routing mode `tunnel`. Load balancers can have targets of type IP, which can either be a public or private (vSwitch) IP of a Robot server ([API reference](https://docs.hetzner.cloud/#load-balancer-actions-add-target)).

As a result, the annotation `load-balancer.hetzner.cloud/use-private-ip` can be set, if the Robot server is connected to a private network and its IP is of type [InternalIP](https://kubernetes.io/docs/reference/node/node-status/#addresses).

## Configuration

To configure this, enable Robot support as outlined in the [robot setup guide](./robot.md). Since the HCCM needs to fetch network data, provide the network ID using the `HCLOUD_NETWORK` environment variable. To prevent the HCCM from making other network changes, disable networking in the Helm chart and set `HCLOUD_NETWORK_ROUTES_ENABLED=false` to turn off the routes controller. Use the following snippet as a reference.

```yaml
networking:
  enabled: false

env:
  HCLOUD_NETWORK:
    valueFrom:
      secretKeyRef:
        name: hcloud
        key: network

  HCLOUD_NETWORK_ROUTES_ENABLED:
    value: "false"

  HCLOUD_TOKEN:
    valueFrom:
      secretKeyRef:
        name: hcloud
        key: token

  ROBOT_USER:
    valueFrom:
      secretKeyRef:
        name: hcloud
        key: robot-user
        optional: true

  ROBOT_PASSWORD:
    valueFrom:
      secretKeyRef:
        name: hcloud
        key: robot-password
        optional: true
```
