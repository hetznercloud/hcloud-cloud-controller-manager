#!/usr/bin/env bash

: "${TEMPLATES_DIR:=./deploy}"

VERSION="$1"

if [[ -z $VERSION ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

cat chart/Chart.yaml | sed -e "s/version: .*/version: $VERSION/" > chart/Chart.yaml.new && mv chart/Chart.yaml{.new,}
helm template chart > deploy/ccm.yaml
helm template chart --set networking.enabled=true > deploy/ccm-networks.yaml

