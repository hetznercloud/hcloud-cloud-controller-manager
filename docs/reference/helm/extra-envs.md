# Helm - Extra Environment Variables

You can define extra environment variables for the HCCM. Both Kubernetes formats are supported: `value` and `valueFrom`. The `valueFrom` field can reference multiple sources such as ConfigMaps and Secrets, but also supports other options. For more details, see the Kubernetes documentation on [ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/#using-configmaps-as-environment-variables) and [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables).

If you want credential hot reloading, do not provide `HCLOUD_TOKEN`, `ROBOT_USER`, or `ROBOT_PASSWORD` via regular environment variables or `valueFrom.secretKeyRef`. Hot reloading only works when these credentials are read from files via `HCLOUD_TOKEN_FILE`, `ROBOT_USER_FILE`, and `ROBOT_PASSWORD_FILE`, backed by a mounted Secret volume. Kubernetes updates mounted Secret files, but it does not update the environment of a running container.

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

Example for file-backed credentials with hot reloading:

```yaml
env:
  HCLOUD_TOKEN: null
  ROBOT_USER: null
  ROBOT_PASSWORD: null
  HCLOUD_TOKEN_FILE:
    value: /etc/hetzner/token
  ROBOT_USER_FILE:
    value: /etc/hetzner/robot-user
  ROBOT_PASSWORD_FILE:
    value: /etc/hetzner/robot-password
```
