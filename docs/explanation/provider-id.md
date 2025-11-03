# Provider ID

The provider ID is a unique identifier of a machine assigned to a Kubernetes Node object. It enables cloud providers to link Kubernetes Nodes with their underlying infrastructure. In Kubernetes, this ID is specified under `spec.providerID` in the Node specification. The Hetzner Cloud Controller Manager sets this value during node initialization.

In the Hetzner ecosystem, the following provider ID formats are used:

- **Hetzner Cloud Server**: `hcloud://<server-id>`
- **Robot Server**: `hrobot://<robot-id>` (default)
- **Legacy Syself Robot Server**: `hcloud://bm-<robot-id>`
  - This format was previously used by the Syself Fork (for [Cluster-API Provider
    Hetzner](https://github.com/syself/cluster-api-provider-hetzner/) and can be enabled via the
    `ROBOT_PROVIDER_ID_SYSELF_FORMAT` environment variable.

## Configuration

For Robot (bare-metal) servers, you can choose between two ProviderID formats:

- **Default format** (`hrobot://<robot-id>`): This is the current standard format used by default.
- **Syself format** (`hcloud://bm-<robot-id>`): This format can be enabled by setting the environment variable `ROBOT_PROVIDER_ID_SYSELF_FORMAT=true`.

You can have a look at your provider IDs with the following command:

```bash
kubectl get nodes -o=custom-columns='Node Name:metadata.name,Provider ID:spec.providerID'
```
