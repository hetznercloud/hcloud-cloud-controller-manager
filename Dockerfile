FROM alpine:3.20.3
RUN apk add --no-cache ca-certificates bash
COPY hetzner-cloud-controller-manager /bin/hetzner-cloud-controller-manager
LABEL org.opencontainers.image.source https://github.com/syself/hetzner-cloud-controller-manager
ENTRYPOINT ["/bin/hetzner-cloud-controller-manager"]
