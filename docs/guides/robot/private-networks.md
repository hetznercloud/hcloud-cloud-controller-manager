# Attach Load Balancers to Robot Private IPs

With the v1.24.0 release we introduced the option to configure Internal IPs for Robot servers. This allows creating a cluster with private networks and a mixture of Robot and Cloud servers. Using the routing feature of private networks is not supported, so this requires a CNI plugin with encapsulation methods, such as Cilium with routing mode `tunnel`. Load Balancers can have targets of type IP, which can either be a public or private (vSwitch) IP of a Robot server ([API reference](https://docs.hetzner.cloud/reference/cloud#load-balancer-actions-add-target)).

As a result, the annotation `load-balancer.hetzner.cloud/use-private-ip` can be set, if the Robot server is connected to a private network and its IP is of type [InternalIP](https://kubernetes.io/docs/reference/node/node-status/#addresses).

## Prerequisite

Enable Robot support as outlined in the [Robot setup guide](TODO). As mentioned there, for a Robot server we pass along configured InternalIPs, that do not appear as an ExternalIP and are within the configured address family. Check with `kubectl get nodes -o json | jq ".items.[].status.addresses"` if you have configured an InternalIP.

## Configuration

Since the HCCM needs to fetch network data, provide the network ID using the `HCLOUD_NETWORK` environment variable. To disable the Routes controller, which is incompatible with vSwitches, disable networking in the Helm chart and set `HCLOUD_NETWORK_ROUTES_ENABLED=false`. Use the following snippet as a reference.

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
