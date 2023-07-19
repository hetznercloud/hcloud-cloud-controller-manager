#!/usr/bin/env bash
set -ueo pipefail

# Template the chart with pre-built values to get the legacy deployment files
helm_template='helm template chart --namespace kube-system --set selectorLabels."app\.kubernetes\.io/name"=null,selectorLabels."app\.kubernetes\.io/instance"=null,selectorLabels.app=hcloud-cloud-controller-manager'
eval $helm_template > deploy/ccm.yaml
eval $helm_template --set networking.enabled=true > deploy/ccm-networks.yaml
