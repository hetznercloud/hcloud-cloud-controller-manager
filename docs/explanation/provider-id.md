# Provider ID

The provider ID is a unique identifier of a machine assigned to a Kubernetes Node object. It enables cloud providers to link Kubernetes Nodes with their underlying infrastructure. In Kubernetes, this ID is specified under `spec.providerID` in the Node specification. The Hetzner Cloud Controller Manager sets this value during node initialization.

In the Hetzner ecosystem, the following provider ID formats are used:

- **Hetzner Cloud Server**: `hcloud://<server-id>`
- **Robot Server**: `hrobot://<robot-id>`
- **Legacy Syself Robot Server**: `hcloud://bm-<robot-id>`
  - This format is no longer used for new nodes but remains for backward compatibility.

You can have a look at your provider IDs with the following command:

```bash
kubectl get nodes -o=custom-columns='Node Name:metadata.name,Provider ID:spec.providerID'
```
