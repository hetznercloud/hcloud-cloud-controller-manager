#!/usr/bin/env bash

set -e
set -o pipefail

CHART_REPO_REMOTE=${CHART_REPO_REMOTE:-"https://github.com/hetznercloud/helm-charts.git"}
CHART_REPO_BRANCH=${CHART_REPO_BRANCH:-"main"}

CHART_FILE="$1"

if [[ -z "$CHART_FILE" ]]; then
  echo "Usage: $0 <name-of-chart.tgz>"
  exit 1
fi

TMP_DIR=$(mktemp --directory hccm-chart-repo.XXXXX)

git clone --depth 1 -b "${CHART_REPO_BRANCH}" "${CHART_REPO_REMOTE}" "${TMP_DIR}"

cp "${CHART_FILE}" "${TMP_DIR}/"

pushd "${TMP_DIR}"

# Setup git-lfs
git lfs install --local

# commit & push
git add "${CHART_FILE}"
git commit -m "feat: add ${CHART_FILE}"
git push

popd
rm -rf "${TMP_DIR}"
