FROM alpine:3.20
RUN apk add --no-cache ca-certificates bash
COPY hcloud-cloud-controller-manager /bin/hcloud-cloud-controller-manager
ENTRYPOINT ["/bin/hcloud-cloud-controller-manager"]
