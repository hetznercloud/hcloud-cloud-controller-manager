# Configuration reference

This page references the different configurations for the hcloud-cloud-controller-manager.

## Extra Environment Variables

Extra environment variables can be set via the `env` Helm value. The well-known Kubernetes formats `value` and `valueFrom` are supported.

```yaml
env:
  ROBUT_USER:
    value: "<robot-user>"
```

```yaml
env:
  ROBUT_USER:
    valueFrom:
      secretKeyRef:
        name: hcloud
        key: robot-user
        optional: true
```
