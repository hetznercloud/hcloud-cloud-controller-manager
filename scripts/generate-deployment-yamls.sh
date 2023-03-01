#!/usr/bin/env bash

: "${TEMPLATES_DIR:=./deploy}"

VERSION="$1"

if [[ -z $VERSION ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

cat chart/Chart.yaml | sed -e "s/version: .*/version: $VERSION/" > chart/Chart.yaml.new && mv chart/Chart.yaml{.new,}
helm template chart > deploy/ccm.yaml

for x in "$TEMPLATES_DIR"/*.yaml.tmpl; do
    outdir="$(command dirname "$x")"/gen
    file="$(command basename "$x")"

    command mkdir -p "$outdir"
    command sed "s/__VERSION__/$VERSION/" "$x" > "$outdir"/"${file%%.tmpl}"
done
