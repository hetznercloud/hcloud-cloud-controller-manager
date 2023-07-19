#!/usr/bin/env bash

: "${TEMPLATES_DIR:=./deploy}"

VERSION="$1"

if [[ -z $VERSION ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# Update version
sed -e "s/version: .*/version: $VERSION/" --in-place chart/Chart.yaml

# Template the chart with pre-built values to get the legacy deployment files
helm_template='helm template chart --namespace kube-system --set selectorLabels."app\.kubernetes\.io/name"=null,selectorLabels."app\.kubernetes\.io/instance"=null,selectorLabels.app=hcloud-cloud-controller-manager'
eval $helm_template > deploy/ccm.yaml
eval $helm_template --set networking.enabled=true > deploy/ccm-networks.yaml

# Package the chart for publishing
helm package chart
