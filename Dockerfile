FROM --platform=linux/amd64 golang:1.22.0-alpine AS builder

WORKDIR /builder
COPY ./ /builder

RUN go build -o hcloud-cloud-controller-manager main.go

FROM --platform=linux/amd64 alpine:3.20
RUN apk add --no-cache ca-certificates bash
COPY --from=builder /builder/hcloud-cloud-controller-manager /bin/

