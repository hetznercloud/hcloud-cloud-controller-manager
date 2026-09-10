# Migrating from [`syself/hetzner-cloud-controller-manager`](https://github.com/syself/hetzner-cloud-controller-manager)

If you have previously used the Hetzner Cloud Controller Manager by Syself, you can migrate to hcloud-cloud-controller-manager. We have tried to keep the configuration & features mostly the same and backwards compatible, but there are some changes you need to be aware of.

## Configuration

### Secret Name

The secret is called `hcloud` in hcloud-cloud-controller-manager, while it was called `hetzner` before. Make sure to create the new secret before migrating your deployment.

### Enable Robot Support

It is now required to explicitly enable support for Robot features. This is done by setting the environment variable `ROBOT_ENABLED=true` on the container, or by setting the value `robot.enabled: true` in the Helm Chart.

## Feature & behaviour changes

### Provider ID

The format of the Provider ID changed from `hcloud://bm-$SERVER_NUMBER` to `hrobot://$SERVER_NUMBER`. For compatibility, we still read from the `hcloud://bm-` prefix, but any new nodes will have the `hrobot://` prefix.

If you read from this value, you should amend your parsing for the new format.

### Load Balancer Targets

In previous versions and the Syself Fork, Robot Targets of the Load Balancer are left alone if Robot support is not enabled.

This was changed, we now remove any Robot Server targets from the Load Balancer if Robot support is not enabled.
