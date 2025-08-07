# Helm - Extra Environment Variables

Extra environment variables can be set via the `env` Helm value. The well-known Kubernetes formats `value` and `valueFrom` are supported.

```yaml
env:
  ROBOT_USER:
    value: "<robot-user>"
```

```yaml
env:
  ROBOT_USER:
    valueFrom:
      secretKeyRef:
        name: hcloud
        key: robot-user
        optional: true
```
