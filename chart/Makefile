RELEASE_NAMESPACE=--namespace kube-system
VALUES = $(wildcard *-values.yaml)
SNAPSHOTS = $(VALUES:%-values.yaml=.snapshots/%.yaml)

all: lint snapshots deploy-manifests

# ----- Snapshots -----

.PHONY: snapshots-clean
snapshots-clean:
	rm -f .snapshots/*

.snapshots/default.yaml:
	helm template hcloud-hccm . $(RELEASE_NAMESPACE) > "$@"

.snapshots/%.yaml: %-values.yaml
	helm template hcloud-hccm . $(RELEASE_NAMESPACE) -f "$<" > "$@"

.PHONY: snapshots
snapshots: snapshots-clean .snapshots/*

# ----- Deploy Manifests -----

SELECTOR_OVERRIDES = \
	--set selectorLabels."app\.kubernetes\.io/name"=null \
	--set selectorLabels."app\.kubernetes\.io/instance"=null \
	--set selectorLabels."app"=hcloud-cloud-controller-manager

.PHONY: deploy-manifests-clean
deploy-manifests-clean:
	rm -f ../deploy/ccm.yaml ../deploy/ccm-networks.yaml

../deploy/ccm.yaml:
	helm template . \
		$(RELEASE_NAMESPACE) \
		$(SELECTOR_OVERRIDES) \
		> ../deploy/ccm.yaml

../deploy/ccm-networks.yaml:
	helm template . \
		$(RELEASE_NAMESPACE) \
		$(SELECTOR_OVERRIDES) \
		--set networking.enabled=true \
		> ../deploy/ccm-networks.yaml

.PHONY: deploy-manifests
deploy-manifests: deploy-manifests-clean ../deploy/*

# ----- Lint -----

.PHONY: lint
lint:
	helm lint .
