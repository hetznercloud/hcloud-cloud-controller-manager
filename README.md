# Kubernetes Cloud Controller Manager for Hetzner Cloud

[![e2e tests](https://github.com/hetznercloud/hcloud-cloud-controller-manager/actions/workflows/test_e2e.yml/badge.svg)](https://github.com/hetznercloud/hcloud-cloud-controller-manager/actions/workflows/test_e2e.yml)
[![Codecov](https://codecov.io/github/hetznercloud/hcloud-cloud-controller-manager/graph/badge.svg?token=Q7pbOoyVpj)](https://codecov.io/github/hetznercloud/hcloud-cloud-controller-manager/tree/main)

The Hetzner Cloud [cloud-controller-manager](https://kubernetes.io/docs/concepts/architecture/cloud-controller/) integrates your Kubernetes cluster with the Hetzner Cloud & Robot APIs.

## Docs

- :rocket: See the [quick start guide](docs/guides/quickstart.md) to get you started.
- :book: See the [configuration reference](docs/reference/README.md) for the available configuration.

For more information, see the [documentation](docs/).

## Development

### Setup a development environment

To set up a development environment, make sure you installed the following tools:

- [tofu](https://opentofu.org/)
- [k3sup](https://github.com/alexellis/k3sup)
- [docker](https://www.docker.com/)
- [skaffold](https://skaffold.dev/)

1. Configure a `HCLOUD_TOKEN` in your shell session.

> [!WARNING]
> The development environment runs on Hetzner Cloud servers which will induce costs.

2. Deploy the development cluster:

```sh
make -C dev up
```

3. Load the generated configuration to access the development cluster:

```sh
source dev/files/env.sh
```

4. Check that the development cluster is healthy:

```sh
kubectl get nodes -o wide
```

5. Start developing hcloud-cloud-controller-manager in the development cluster:

```sh
skaffold dev
```

On code change, skaffold will rebuild the image, redeploy it and print all logs.

⚠️ Do not forget to clean up the development cluster once are finished:

```sh
make -C dev down
```

### Run the unit tests

To run the unit tests, make sure you installed the following tools:

- [Go](https://go.dev/)

1. Run the following command to run the unit tests:

```sh
go test ./...
```

### Run the kubernetes e2e tests

Before running the e2e tests, make sure you followed the [Setup a development environment](#setup-a-development-environment) steps.

1. Run the kubernetes e2e tests using the following command:

```sh
source dev/files/env.sh
go test ./tests/e2e -tags e2e -v
```

### Development with Robot

If you want to work on the Robot support, you need to make some changes to the above setup.

This requires that you have a Robot Server in the same account you use for the development. The server needs to be setup with the Ansible Playbook `dev/robot/install.yml` and configured in `dev/robot/install.yml`.

1. Set these environment variables:

```shell
export ROBOT_ENABLED=true

export ROBOT_USER=<Your Robot User>
export ROBOT_PASSWORD=<Your Robot Password>
```

2. Continue with the environment setup until you reach the `skaffold` step. Run `skaffold dev --profile=robot` instead.

3. We have another suite of tests for Robot. You can run these with:

```sh
go test ./tests/e2e -tags e2e,robot -v
```

## License

Apache License, Version 2.0
