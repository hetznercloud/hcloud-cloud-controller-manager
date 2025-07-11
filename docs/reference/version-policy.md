# Version Policy

We aim to support the latest three versions of Kubernetes. When a Kubernetes
version is marked as _End Of Life_, we will stop support for it and remove the
version from our CI tests. This does not necessarily mean that the
Cloud Controller Manager does not still work with this version. We will
not fix bugs related only to an unsupported version.

Current Kubernetes Releases: https://kubernetes.io/releases/

### With Networks support

| Kubernetes | Cloud Controller Manager |                                                                                             Deployment File |
| ---------- | -----------------------: | ----------------------------------------------------------------------------------------------------------: |
| 1.33       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.32       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.31       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.30       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm-networks.yaml |
| 1.29       |                  v1.25.1 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.25.1/ccm-networks.yaml |
| 1.28       |                  v1.20.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.20.0/ccm-networks.yaml |
| 1.27       |                  v1.20.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.20.0/ccm-networks.yaml |
| 1.26       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm-networks.yaml |
| 1.25       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm-networks.yaml |
| 1.24       |                  v1.17.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.17.2/ccm-networks.yaml |
| 1.23       |                  v1.13.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.13.2/ccm-networks.yaml |

### Without Networks support

| Kubernetes | Cloud Controller Manager |                                                                                    Deployment File |
| ---------- | -----------------------: | -------------------------------------------------------------------------------------------------: |
| 1.33       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.32       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.31       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.30       |                   latest |  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml |
| 1.29       |                  v1.25.1 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.25.1/ccm.yaml |
| 1.28       |                  v1.20.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.20.0/ccm.yaml |
| 1.27       |                  v1.20.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.20.0/ccm.yaml |
| 1.26       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm.yaml |
| 1.25       |                  v1.19.0 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.19.0/ccm.yaml |
| 1.24       |                  v1.17.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.17.2/ccm.yaml |
| 1.23       |                  v1.13.2 | https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.13.2/ccm.yaml |
