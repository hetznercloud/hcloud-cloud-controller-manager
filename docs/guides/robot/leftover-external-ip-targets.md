# Check for leftover ExternalIP targets of Robot servers

When `use-private-ip` is enabled, the Load Balancer should only target the `InternalIP` of a Robot server. Older versions kept the ExternalIP target as well. After the upgrade, these leftover ExternalIP targets are removed, so the `InternalIP` target must be healthy.

Make sure `kubectl` points to your cluster and `hcloud` to the project of your Load Balancers.

## 1. List the addresses of your Robot nodes

```bash
kubectl get nodes -o json | jq -r '
  .items[]
  | select(.spec.providerID | test("^(hrobot://|hcloud://bm-)"))
  | "\(.metadata.name): " + ([.status.addresses[] | select(.type != "Hostname") | "\(.type)=\(.address)"] | join(" "))
'
```

Example output:

```
robot-1: ExternalIP=1.1.1.1 InternalIP=10.0.0.1
```

## 2. List the IP targets of your Load Balancers

```bash
hcloud load-balancer list -o json | jq -r '
  .[]
  | .name as $lb
  | .targets[]
  | select(.type == "ip")
  | "\($lb): \(.ip.ip) " + ([.health_status[] | "port \(.listen_port)=\(.status)"] | join(" "))
'
```

Example output:

```
my-lb: 1.1.1.1 port 80=healthy
my-lb: 10.0.0.1 port 80=unhealthy
```

## 3. Compare

You are affected if a Load Balancer has a target for both the ExternalIP and the `InternalIP` of the same Robot node. The ExternalIP target will be removed after the upgrade. If the `InternalIP` target is not `healthy`, the node stops receiving traffic from the Load Balancer. In this case, fix your private network setup (e.g. the vSwitch connection) before upgrading.

In the example above, `robot-1` is affected: `1.1.1.1` will be removed and `10.0.0.1` is unhealthy.

## 4. Check for InternalIPs that can't be added

If the `InternalIP` of a Robot node can't be added as a target, the Load Balancer only has the ExternalIP target of this node, so it does not show up in the comparison above. This happens for example if the `InternalIP` is not part of a vSwitch subnet of the Load Balancer's network. After the upgrade, the ExternalIP target is removed before the `InternalIP` target is added. The node no longer receives traffic and the reconciliation fails until the configuration is fixed.

The reconciliation already fails before the upgrade, so check the events of your Services:

```bash
kubectl get events -A --field-selector reason=SyncLoadBalancerFailed | grep ip_not_in_vswitch_subnet
```

Alternatively, list the vSwitch subnets of your network and make sure that every `InternalIP` from step 1 is part of one of them:

```bash
hcloud network describe <network> -o json | jq -r '.subnets[] | select(.type == "vswitch") | .ip_range'
```
