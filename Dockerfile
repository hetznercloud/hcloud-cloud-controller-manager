FROM alpine:3.17
RUN apk add --no-cache ca-certificates bash
COPY hetzner-cloud-controller-manager /bin/hetzner-cloud-controller-manager
ENTRYPOINT ["/bin/hetzner-cloud-controller-manager"]
