FROM golang:1.24 as builder

WORKDIR /build

ADD go.mod go.sum /build/
RUN go mod download

ADD . /build/
RUN ls -al

ARG CGO_ENABLED=0
# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
ARG GO_BUILDFLAGS
ENV GOCACHE=/cache
RUN --mount=type=cache,target="/cache" go build -gcflags="$SKAFFOLD_GO_GCFLAGS" $GO_BUILDFLAGS -o hcloud-cloud-controller-manager.bin github.com/hetznercloud/hcloud-cloud-controller-manager

FROM alpine:3.22

RUN apk add --no-cache \
    bash \
    ca-certificates

COPY --from=builder /build/hcloud-cloud-controller-manager.bin /bin/hcloud-cloud-controller-manager
ENTRYPOINT ["/bin/hcloud-cloud-controller-manager"]
