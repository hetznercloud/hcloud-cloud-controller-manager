# Helm - Extra Environment Variables

You can define extra environment variables for the HCCM. Both Kubernetes formats are supported: `value` and `valueFrom`. The `valueFrom` field can reference multiple sources such as ConfigMaps and Secrets, but also supports other options. For more details, see the Kubernetes documentation on [ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/#using-configmaps-as-environment-variables) and [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables).

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
