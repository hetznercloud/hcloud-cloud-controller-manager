# Clusters with Robot Servers

## Quickstart

Prerequisites:

- Running Cluster
- Kubectl
- Helm

0. Make sure that you start all Kubelets in the cluster with `--cloud-provider=external`.

1. Export your credentials and create a secret:

   ```bash
    export HCLOUD_TOKEN=your-hcloud-token
    export ROBOT_USER=your-robot-user-name
    export ROBOT_PASSWORD=your-robot-password
   kubectl -n kube-system create secret generic hcloud --from-literal=token=$HCLOUD_TOKEN --from-literal=robot-user=$ROBOT_USER --from-literal=robot-password=$ROBOT_PASSWORD
   ```

2. Install the Helm Chart:

   ```bash
   helm repo add hcloud https://charts.hetzner.cloud
   helm repo update hcloud
   helm install hcloud/hcloud-cloud-controller-manager --set robot.enabled=true
   ```

You should now see that the Robot Server was initialized and [some labels](#node-controller) added.

## Features

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
- Provider ID
  - We set the field `Node.spec.providerID` to identify the Robot server after the initial adoption.
  - The format is `hrobot://$SERVER_NUMBER`, but we can also read from the deprecated format used by [syself/hetzner-cloud-controller-manager](https://github.com/syself/hetzner-cloud-controller-manager): `hcloud://bm-$SERVER_NUMBER`
- Addresses
  - We add the Hostname and (depending on the configuration and availability) the IPv4 and IPv6 addresses of the server in `Node.status.addresses`.
  - For the IPv6 address we use the first address in the Network -> For the network `2a01:f48:111:4221::` we add the address `2a01:f48:111:4221::1`.
  - Private IPs in a vSwitch are not supported.

### Node Lifecycle Controller

The Node Lifecycle Controller is responsible for updating the shutdown status of Nodes & deleting the Kubernetes Node object if the corresponding server is removed.

Both are generally supported. The shutdown status can only be detected if the Robot Server supports this.

### Service Controller (Load Balancers)

The service controller watches Services with `type: LoadBalancer` and creates Cloud Load Balancers for them. By default, all Kubernetes Nodes including Robot servers are added as targets to the Load Balancer. Check out the [Load Balancer Documentation](./load_balancers.md) for more details.

### Unsupported

#### Routes & Private Networks

Adding support for Routing Pod CIDRs through the (Cloud) Networks and (Robot) vSwitches is not currently supported. You will need to use your own CNI for this.

> If you are interested in this, we are looking for contributors to help design & implement this.

## Requirements

### Identifying the correct Server

When a new Node joins the cluster, we first need to figure out which Robot (or Cloud) Server matches this node. We primarily try to match this through the Node Name and the Name of the server in Robot. If you use Kubeadm, the Node Name by default is the Hostname of the server.

_This means that by default, your **Hostname** needs to be the **name of the server in Robot**_. If this does not match, we can not properly match the two entities. Once we have made this connection, we save the Robot Server Number to the field `spec.providerId` on the Node, and use this identifier for any further processing.

If you absolutely need to use different names in Robot & Hostname, you can also configure the Provider ID yourself. This can be done on the `kubelet` through the flag [`--provider-id`](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/). You need to follow the format `hrobot://$SERVER_NUMBER` when setting this. If this format is not followed exactly we can not process this node.

## Config Options

### Credentials

You need to add your Robot credentials into the secret `hcloud`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: hcloud
  namespace: kube-system
type: Opaque
stringData:
  token: your-hcloud-token
  robot-user: your-robot-user-name
  robot-password: your-robot-password
```

If you only plan to use a single Robot server, you can also use an "Admin login" (see the `Admin login` tab on the [server administration page](https://robot.hetzner.com/server)) for this server instead of the account credentials.

Then you can enable the Robot Support through the environment variable `ROBOT_ENABLED=true` or the Helm Chart value `robot.enabled: true`.

You will also need to [disable Network support](#routes--private-networks) through the Helm Chart value `network.enabled: false`. If you use plain Kubernetes manifests, make sure you use the `ccm.yaml` and not the `ccm-network.yaml`.


## Migrating from [`syself/hetzner-cloud-controller-manager`](https://github.com/syself/hetzner-cloud-controller-manager)

If you have previously used the Hetzner Cloud Controller Manager by Syself, you can migrate to hcloud-cloud-controller-manager. We have tried to keep the configuration & features mostly the same and backwards compatible, but there are some changes you need to be aware of.

### Configuration

#### Secret Name

The secret is called `hcloud` in hcloud-cloud-controller-manager, while it was called `hetzner` before. Make sure to create the new secret before migrating your deployment.

#### Enable Robot Support

It is now required to explicitly enable support for Robot features. This is done by setting the environment variable `ROBOT_ENABLED=true` on the container, or by setting the value `robot.enabled: true` in the Helm Chart.

### Feature & behaviour changes

#### Provider ID

The format of the Provider ID changed from `hcloud://bm-$SERVER_NUMBER` to `hrobot://$SERVER_NUMBER`. For compatibility, we still read from the `hcloud://bm-` prefix, but any new nodes will have the `hrobot://` prefix.

If you read from this value, you should amend your parsing for the new format.

#### Load Balancer Targets

In previous versions and the Syself Fork, Robot Targets of the Load Balancer are left alone if Robot support is not enabled.

This was changed, we now remove any Robot Server targets from the Load Balancer if Robot support is not enabled.
