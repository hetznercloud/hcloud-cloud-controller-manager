export CGO_ENABLED:=0

VERSION=$(shell ./scripts/git-version.bash)
DOCKER_REPO=hetznercloud/hcloud-cloud-controller-manager

all: build

build: clean bin/hcloud-cloud-controller-manager

bin/%:
	@go build -o bin/$* .

container: build
	docker build -t $(DOCKER_REPO):$(VERSION) .

release-container:
	docker push $(DOCKER_REPO):$(VERSION)

test:
	@./scripts/test.bash

clean:
	@rm -rf bin/*

.PHONY: all build clean test container release-container
