#!/usr/bin/env bash
set -ueo pipefail

# Template the chart with pre-built values to get the legacy deployment files
template() {
  helm template chart \
    --namespace kube-system \
    --set selectorLabels."app\.kubernetes\.io/name"=null \
    --set selectorLabels."app\.kubernetes\.io/instance"=null \
    --set selectorLabels."app"=hcloud-cloud-controller-manager \
    "$@"
}

template > deploy/ccm.yaml
template --set networking.enabled=true > deploy/ccm-networks.yaml
