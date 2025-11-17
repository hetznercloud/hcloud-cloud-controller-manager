RELEASE_NAMESPACE=--namespace kube-system

all: lint snapshots templates

# ----- Snapshots -----

default-snapshot: .snapshots/default.yaml
	helm template hcloud-hccm . \
		$(RELEASE_NAMESPACE) | \
		grep -v helm.sh/chart \
		> .snapshots/default.yaml

full-snapshot: .snapshots/full.daemonset.yaml
	helm template hcloud-hccm . \
		$(RELEASE_NAMESPACE) \
		-f .snapshots/full.values.daemonset.yaml \
		| grep -v helm.sh/chart \
		> .snapshots/full.daemonset.yaml

snapshots-diff: default-snapshot full-snapshot
	git diff --exit-code -- .snapshots/ || echo "WARNING: git diff found in chart/.snapshots/; run make -C chart snapshots"

snapshots: snapshots-diff

# ----- Templates -----
#
SELECTOR_OVERRIDES = \
	--set selectorLabels."app\.kubernetes\.io/name"=null \
	--set selectorLabels."app\.kubernetes\.io/instance"=null \
	--set selectorLabels."app"=hcloud-cloud-controller-manager

default-template: ../deploy/ccm.yaml
	helm template . \
		$(RELEASE_NAMESPACE) \
		$(SELECTOR_OVERRIDES) \
		> ../deploy/ccm.yaml

networking-template: ../deploy/ccm-networks.yaml
	helm template . \
		$(RELEASE_NAMESPACE) \
		$(SELECTOR_OVERRIDES) \
		--set networking.enabled=true \
		> ../deploy/ccm-networks.yaml

templates-diff: default-template networking-template
	git diff --exit-code -- ../deploy/ || echo "WARNING: git diff found in deploy/; run make -C chart templates"

templates: templates-diff

# ----- Lint -----

.PHONY: lint
lint:
	helm lint .
