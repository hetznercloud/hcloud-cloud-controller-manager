# Hybrid Clusters

Clusters with nodes that are neither Hetzner Cloud nor Robot servers (e.g. on-premise or other cloud providers) are not supported.

HCCM implements the interfaces of the upstream [`cloudprovider`](https://pkg.go.dev/k8s.io/cloud-provider) library, which also contains the controller logic. These controllers assume that all nodes in the cluster are managed by a single cloud controller manager. HCCM therefore reports errors and warnings for nodes with an unknown [provider ID](provider-id.md). Ignoring them would cause the upstream controllers to treat them as non-existent and act accordingly, e.g. by deleting the Node.

We will not add options to ignore such nodes, e.g. by provider ID prefix, until this is supported upstream.

Progress for supporting different prefixes in the node controllers is tracked in [KEP-5778](https://github.com/kubernetes/enhancements/issues/5778).
