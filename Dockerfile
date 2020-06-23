FROM golang:1.14 as builder
WORKDIR /maschine-controller/src
ADD . .

FROM alpine:3.11
RUN apk add --no-cache ca-certificates bash
COPY --from=builder /maschine-controller/src/hcloud-maschine-controller.bin /bin/hcloud-cloud-controller-manager
ENTRYPOINT ["/bin/hcloud-cloud-controller-manager"]
