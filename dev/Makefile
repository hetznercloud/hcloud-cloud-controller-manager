SHELL = bash
.ONESHELL:

ENV ?= dev
K3S_CHANNEL ?= stable

env.auto.tfvars:
	@echo 'name         = "$(ENV)"' > "$@"
	@echo 'hcloud_token = "$(HCLOUD_TOKEN)"' >> "$@"
	@echo 'k3s_channel  = "$(K3S_CHANNEL)"' >> "$@"

.terraform:
	tofu init

validate: .terraform
	tofu validate

up: .terraform env.auto.tfvars
	tofu apply -auto-approve
	$(MAKE) port-forward

down: .terraform env.auto.tfvars
	tofu destroy -auto-approve

port-forward:
	source files/env.sh
	bash files/registry-port-forward.sh

clean:
	rm -Rf files/ .terraform/ terraform.tfstate* env.auto.tfvars
