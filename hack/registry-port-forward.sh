#!/usr/bin/env bash
set -ue -o pipefail
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

{
until kubectl -n kube-system --timeout=30s rollout status deployment/registry-docker-registry >/dev/null 2>&1; do sleep 1; done
old_pid=$(cat $SCRIPT_DIR/.reg-pf 2>/dev/null || true)
if [[ -n "$old_pid" ]]; then
  echo "killing old port-forward with PID $old_pid"
  kill $old_pid || true
fi

nohup kubectl port-forward -n kube-system svc/registry-docker-registry 30666:5000 >$SCRIPT_DIR/.reg-pf.out 2>$SCRIPT_DIR/.reg-pf.err &
} >&2

echo $! > $SCRIPT_DIR/.reg-pf
