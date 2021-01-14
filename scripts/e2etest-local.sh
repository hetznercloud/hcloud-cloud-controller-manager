#!/usr/bin/env bash

set -e

function test_k8s_version() {
    if [[ -z "$1" ]]; then
        echo "Usage: $0 <k8s-version>"
        return 1
    fi

    export K8S_VERSION="$1"

    echo "Testing $K8S_VERSION without network support"
    export USE_NETWORKS="no"
    if ! go test -count=1 -v -timeout 60m ./e2etests; then
        return 2
    fi

    echo
    echo
    echo "Testing $K8S_VERSION with network support"
    export USE_NETWORKS="yes"
    if ! go test -count=1 -v -timeout 60m ./e2etests; then
        return 2
    fi
}

if [[ -z "$HCLOUD_TOKEN" ]]; then
    echo "HCLOUD_TOKEN not set! Aborting tests."
    exit 1
fi

K8S_VERSIONS=("k8s-1.18.12" "k8s-1.19.4" "k8s-1.20.0" "k3s-v1.20.0+k3s2")
for v in "${K8S_VERSIONS[@]}"; do
    test_k8s_version "$v"
done
