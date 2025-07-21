# Address Family

To control the address family used when initializing a node, the `HCLOUD_INSTANCES_ADDRESS_FAMILY` environment variable can be set to `ipv4`, `ipv6` or `dualstack`. By default, the value is set to `ipv4`.

## Configuration via Helm

```yaml
# values.yaml
---
env:
  HCLOUD_INSTANCES_ADDRESS_FAMILY:
    value: "dualstack"
```
