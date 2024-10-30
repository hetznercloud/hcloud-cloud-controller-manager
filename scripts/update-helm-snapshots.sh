#!/usr/bin/env bash
set -ueo pipefail

helm template hcloud-hccm chart \
  --namespace kube-system |
    grep -v helm.sh/chart \
    > chart/.snapshots/default.yaml

helm template hcloud-hccm chart \
  --namespace kube-system \
  -f chart/.snapshots/full.values.daemonset.yaml |
    grep -v helm.sh/chart \
    > chart/.snapshots/full.daemonset.yaml
