#!/usr/bin/env bash

set -eu

diff -u <(echo -n) <(gofmt -d -s $(find . -type f -name '*.go' | grep -v ^./vendor))
go vet ./...
go test ./...
