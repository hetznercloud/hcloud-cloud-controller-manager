SHELL = bash

ENV ?= dev
K3S_CHANNEL ?= stable
ROBOT_ENABLED ?= false

env.auto.tfvars:
	@echo 'name           = "$(ENV)"' > "$@"
	@echo 'hcloud_token   = "$(HCLOUD_TOKEN)"' >> "$@"
	@echo 'k3s_channel    = "$(K3S_CHANNEL)"' >> "$@"
	@echo 'robot_enabled  = "$(ROBOT_ENABLED)"' >> "$@"
	@echo 'robot_user     = "$(ROBOT_USER)"' >> "$@"
	@echo 'robot_password = "$(ROBOT_PASSWORD)"' >> "$@"

.terraform:
	tofu init

validate: .terraform
	tofu validate

up: .terraform env.auto.tfvars
	tofu apply -auto-approve
	$(MAKE) port-forward

down: .terraform env.auto.tfvars
	if test -f files/registry-port-forward.sh; then files/registry-port-forward.sh down; fi
	tofu destroy -auto-approve

port-forward:
	source files/env.sh && files/registry-port-forward.sh up

clean:
	rm -Rf files/ .terraform/ terraform.tfstate* env.auto.tfvars
