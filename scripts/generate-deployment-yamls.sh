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
helm template chart > deploy/ccm.yaml
helm template chart --set networking.enabled=true > deploy/ccm-networks.yaml

# Package the chart for publishing
helm package chart
