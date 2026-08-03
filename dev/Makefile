SHELL = bash

ENV ?= dev
K3S_CHANNEL ?= stable
ROBOT_ENABLED ?= false

# The cluster (infra) and the resources inside of it (k8s) are managed in two
# separate states, and must always be applied in this order.
STATES = infra k8s

infra/env.auto.tfvars:
	@echo 'name          = "$(ENV)"' > "$@"
	@echo 'hcloud_token  = "$(HCLOUD_TOKEN)"' >> "$@"
	@echo 'k3s_channel   = "$(K3S_CHANNEL)"' >> "$@"
	@echo 'robot_enabled = "$(ROBOT_ENABLED)"' >> "$@"

k8s/env.auto.tfvars:
	@echo 'hcloud_token   = "$(HCLOUD_TOKEN)"' > "$@"
	@echo 'robot_enabled  = "$(ROBOT_ENABLED)"' >> "$@"
	@echo 'robot_user     = "$(ROBOT_USER)"' >> "$@"
	@echo 'robot_password = "$(ROBOT_PASSWORD)"' >> "$@"

tfvars: infra/env.auto.tfvars k8s/env.auto.tfvars

%/.terraform:
	tofu -chdir=$* init

init: infra/.terraform k8s/.terraform

validate: init
	for state in $(STATES); do tofu -chdir=$$state validate || exit 1; done

up: init tfvars
	tofu -chdir=infra apply -auto-approve
	tofu -chdir=k8s apply -auto-approve
	$(MAKE) port-forward

down: init tfvars
	if test -f files/registry-port-forward.sh; then files/registry-port-forward.sh down; fi
	# The resources in the cluster live on servers that are deleted right after,
	# a failing teardown (e.g. an unreachable cluster) must not block us.
	-tofu -chdir=k8s destroy -auto-approve
	tofu -chdir=infra destroy -auto-approve

port-forward:
	source files/env.sh && files/registry-port-forward.sh up

clean:
	rm -Rf files/ */.terraform/ */terraform.tfstate* */env.auto.tfvars
