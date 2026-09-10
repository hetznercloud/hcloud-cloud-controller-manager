# IPv6 ExternalIP

We know the IPv6 subnet assigned to a server, but not the address the server actually uses within it. By default, we use the first address of the subnet, e.g. `2001:db8:1234::1` for the subnet `2001:db8:1234::`. To set the address explicitly, annotate the Node with `instance.hetzner.cloud/external-ipv6`:

```bash
kubectl annotate node $NODE_NAME --overwrite \
  instance.hetzner.cloud/external-ipv6=2001:db8:1234::5
```

The annotation takes precedence over the derived address. For [Robot](robot/README.md) servers it is the only source if Robot reports no IPv6 subnet, e.g. when IPv6 is configured on upstream network equipment instead of per server.

The annotation only takes effect once IPv6 is enabled for Node addresses, see [Address Family](address-family.md). If the configured address family does not include IPv6, we ignore the annotation and emit an `IgnoredExternalIPv6` warning event on the Node.

## Troubleshooting

The annotation must contain a global unicast IPv6 address, e.g. `2001:db8:1234::5`. Otherwise the Node fails to initialize, instead of falling back to the derived address, and we emit an `InvalidExternalIPv6` warning event on the Node:

```bash
kubectl describe node $NODE_NAME
```
