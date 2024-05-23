FROM golang:1.21 as builder
WORKDIR /hccm
ADD go.mod go.sum /hccm/
RUN go mod download
ADD . /hccm/
RUN ls -al
# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
RUN CGO_ENABLED=0 go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o hcloud-cloud-controller-manager.bin github.com/hetznercloud/hcloud-cloud-controller-manager

FROM alpine:3.20
RUN apk add --no-cache ca-certificates bash
COPY --from=builder /hccm/hcloud-cloud-controller-manager.bin /bin/hcloud-cloud-controller-manager
ENTRYPOINT ["/bin/hcloud-cloud-controller-manager"]
