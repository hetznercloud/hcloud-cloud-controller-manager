# Robot Support

Most of the features we support for Cloud servers are also supported for Robot servers:

### Node Controller

The Node controller adds information about the server to the Node object. The values are changed from what you usually see in the Robot interface & Webservice to better match the Cloud counterpart.

- Labels
  - `node.kubernetes.io/instance-type`
    - Examples: `AX41` `Server-Auction`
    - We replace any empty spaces with `-` (hyphen)
  - `topology.kubernetes.io/region`
    - Examples: `hel1` `fsn1`
    - We use the lowercase variant of the location to match the Cloud Locations
  - `topology.kubernetes.io/zone`
    - Examples: `hel1-dc5` `fsn1-dc16`
    - We use the lowercase variant of the location to match the Cloud Datacenters
  - `instance.hetzner.cloud/provided-by`
    - Examples: `robot` `cloud`
    - We detect if the node is a Robot server or Cloud VM and set the label accordingly
- Provider ID
  - We set the field `Node.spec.providerID` to identify the Robot server after the initial adoption.
  - The format is `hrobot://$SERVER_NUMBER`, but we can also read from the deprecated format used by [syself/hetzner-cloud-controller-manager](https://github.com/syself/hetzner-cloud-controller-manager): `hcloud://bm-$SERVER_NUMBER`
- Addresses
  - We add the Hostname and (depending on the configuration and availability) the IPv4 and IPv6 addresses of the server in `Node.status.addresses`.
  - For the IPv6 address we use the first address in the Network -> For the network `2a01:f48:111:4221::` we add the address `2a01:f48:111:4221::1`.
  - Automatic reporting of private IPs in a vSwitch to `Node.status.addresses` are not supported.
  - By default, we pass along InternalIPs configured via the kubelet flag `--node-ip`. This can be disabled by setting the environment variable `ROBOT_FORWARD_INTERNAL_IPS` to `false`. It is not allowed to configure the same IP for InternalIP and ExternalIP.

### Node Lifecycle Controller

The Node Lifecycle Controller is responsible for updating the shutdown status of Nodes & deleting the Kubernetes Node object if the corresponding server is removed.

Both are generally supported. The shutdown status can only be detected if the Robot Server supports this.

### Service Controller (Load Balancers)

The service controller watches Services with `type: LoadBalancer` and creates Cloud Load Balancers for them. By default, all Kubernetes Nodes including Robot servers are added as targets to the Load Balancer. Check out the [Load Balancer Documentation](./load_balancers.md) for more details.

### Unsupported

#### Routes

Adding support for Routing Pod CIDRs through the (Cloud) Networks and (Robot) vSwitches is not currently supported. You will need to use your own CNI for this.

## Identifying the correct Server

When a new Node joins the cluster, we first need to figure out which Robot (or Cloud) Server matches this node. We primarily try to match this through the Node Name and the Name of the server in Robot. If you use Kubeadm, the Node Name by default is the Hostname of the server.

_This means that by default, your **Hostname** needs to be the **name of the server in Robot**_. If this does not match, we can not properly match the two entities. Once we have made this connection, we save the Robot Server Number to the field `spec.providerId` on the Node, and use this identifier for any further processing.

If you absolutely need to use different names in Robot & Hostname, you can also configure the Provider ID yourself. This can be done on the `kubelet` through the flag [`--provider-id`](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/). You need to follow the format `hrobot://$SERVER_NUMBER` when setting this. If this format is not followed exactly we can not process this node.

## Credentials

If you only plan to use a single Robot server, you can also use an "Admin login" (see the `Admin login` tab on the [server administration page](https://robot.hetzner.com/server)) for this server instead of the account credentials.
