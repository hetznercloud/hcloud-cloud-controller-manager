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

snapshots: default-snapshot full-snapshot

# ----- Templates -----

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

templates: default-template networking-template

# ----- Lint -----

.PHONY: lint
lint:
	helm lint .
