#!/usr/bin/env bash

: "${TEMPLATES_DIR:=./deploy}"

VERSION="$1"

if [[ -z $VERSION ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

for x in "$TEMPLATES_DIR"/*.yaml.tmpl; do
    outdir="$(command dirname "$x")"/gen
    file="$(command basename "$x")"

    command mkdir -p "$outdir"
    command sed "s/__VERSION__/$VERSION/" "$x" > "$outdir"/"${file%%.tmpl}"
done
