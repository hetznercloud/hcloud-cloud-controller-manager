FROM golang:1.18 as builder

WORKDIR /hccm
ADD . /hccm/
RUN CGO_ENABLED=0 go build -o hcloud-cloud-controller-manager .

FROM alpine:3.12

RUN apk add --no-cache ca-certificates bash
COPY --from=builder /hccm/hcloud-cloud-controller-manager /bin/hcloud-cloud-controller-manager

ENTRYPOINT ["/bin/hcloud-cloud-controller-manager"]
