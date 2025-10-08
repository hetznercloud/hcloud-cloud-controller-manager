#!/usr/bin/env bash

set -e
set -o pipefail

: "${CHART_REPO_REMOTE:="https://github.com/hetznercloud/helm-charts.git"}"
: "${CHART_REPO_BRANCH:="main"}"

CHART_FILE="$1"

if [[ -z "$CHART_FILE" ]]; then
  echo "Usage: $0 <name-of-chart.tgz>"
  exit 1
fi

# Ensures we only publish once, when $1 == $GORELEASER_ARTIFACT_FILE
if [[ -n "${GORELEASER_ARTIFACT_FILE}" && "${GORELEASER_ARTIFACT_FILE}" != "${CHART_FILE}" ]]; then
  echo "skipping artifact: ${GORELEASER_ARTIFACT_FILE}" >&2
  exit 0
fi

TMP_DIR=$(mktemp --directory chart-repo.XXXXX)
# shellcheck disable=SC2064
trap "rm -Rf '$(realpath "$TMP_DIR")'" EXIT

git clone --depth 1 -b "${CHART_REPO_BRANCH}" "${CHART_REPO_REMOTE}" "${TMP_DIR}"

if [[ -f "${TMP_DIR}/${CHART_FILE}" ]]; then
  echo "chart file already exists: ${CHART_FILE}"
  exit 0
fi

mkdir "${TMP_DIR}/new-chart"
cp "${CHART_FILE}" "${TMP_DIR}/new-chart"

pushd "${TMP_DIR}/new-chart"

# Update index
# We use --merge to not update any of the other existing entries in the index file,
# this requires us to put our new chart in a separate dir that only includes the new chart.
helm repo index --merge ../index.yaml .
# Move chart and merged index to root dir
mv -f -- * ..

popd
pushd "${TMP_DIR}"

# Setup git-lfs
git lfs install --local

# commit & push
git add -- index.yaml "${CHART_FILE}"
git commit -m "feat: add ${CHART_FILE}"
git push

popd
