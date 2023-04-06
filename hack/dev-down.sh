#!/usr/bin/env bash
set -ue -o pipefail
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

scope="${SCOPE:-dev}"
scope=${scope//[^a-zA-Z0-9_]/-}
scope_name=hccm-${scope}
label="managedby=hack"

if [[ "${ALL:-}" == "" ]]; then
  label="$label,scope=$scope_name"
  rm -f $SCRIPT_DIR/.ssh-$scope $SCRIPT_DIR/.kubeconfig-$scope
else
  rm -f $SCRIPT_DIR/.ssh* $SCRIPT_DIR/.kubeconfig*
fi

for instance in $(hcloud server list -o noheader -o columns=id -l $label); do
  (
    hcloud server delete $instance
  ) &
done


for key in $(hcloud ssh-key list -o noheader -o columns=name -l $label); do
  (
    hcloud ssh-key delete $key
  ) &
done


for key in $(hcloud network list -o noheader -o columns=name -l $label); do
  (
    hcloud network delete $key
  ) &
done

wait
