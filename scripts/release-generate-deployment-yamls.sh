#!/usr/bin/env bash

set -ueo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"

: "${TEMPLATES_DIR:=./deploy}"

VERSION="$1"

if [[ -z $VERSION ]]; then
  echo "Usage: $0 <version>"
  exit 1
fi

# Update version
sed -e "s/version: .*/version: $VERSION/" --in-place chart/Chart.yaml

"$SCRIPT_DIR"/update-deployment-yamls.sh

# Package the chart for publishing
helm package chart
