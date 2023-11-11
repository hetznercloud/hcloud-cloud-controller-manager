# Clusters with Robot Servers

## Features

Most of the features we support for Cloud servers are also supported for Robot servers:

### Node Controller

The Node controller adds some information about the server to the Node object. This includes:

- `TODO Annotations`

### Node Lifecycle Controller

The Node Lifecycle Controller is responsible for updating the shutdown status of Nodes & deleting the Kubernetes Node object if the corresponding server is removed.

Both are generally supported. The shutdown status can only be detected if the Robot Server supports this.

### Service Controller (Load Balancers)

The service controller watches Services with `type: LoadBalancer` and creates Cloud Load Balancers for them. By default, all Kubernetes Nodes including Robot servers are added as targets to the Load Balancer. Check out the [Load Balancer Documentation](./load_balancers.md) for more details.

### Unsupported

#### Routes & Private Networks

Adding support for Routing Pod CIDRs through the (Cloud) Networks & (Robot) vSwitches is not currently supported. You will need to use your own CNI for this. 

If you are interested in this, we are looking for contributors to help design & implement this.

## Requirements

### Identifying the correct Server

When a new Node joins the cluster, we first need to figure out which Robot (or Cloud) Server matches this node. We primarily try to match this through the Node Name & the Name of the server in Robot. If you use Kubeadm, the Node Name by default is the Hostname of the server.

_This means that by default, your **Hostname** needs to be the same of the **name of the server in Robot**_. If this does not match, we can not properly match the two entities. Once we have made this connection, we save the Robot Server Number to the field `spec.providerId` on the Node, and use this identifier for any further processing.

If you absolutely need to use different names in Robot & Hostname, you can also configure the Provider ID yourself. With Kubeadm you can set the flag `TODO` to specify it manually. You need to follow the format `hrobot://$SERVER_NUMBER` when setting this. If this format is not followed exactly we can not process this node.

## Config Options

## 

## Migrating from syself/hetzner-cloud-controller-manager

If you have previously used the Hetzner Cloud Controller Manager by Syself, you can migrate to hcloud-cloud-controller-manager. We have tried to keep the configuration & features mostly the same and backwards compatible, but you need to make the following changes:

