FROM golang:1.12 as builder
ENV GO111MODULE=on
WORKDIR $GOPATH/src/github.com/hetznercloud/hcloud-cloud-controller-manager
ADD go.mod go.sum ./
RUN go mod download
ADD . .
RUN CGO_ENABLED=0 go build -o /bin/hcloud-maschine-controller  .

FROM alpine:3.9
RUN apk add --no-cache ca-certificates bash
COPY --from=builder /bin/hcloud-maschine-controller /bin/hcloud-cloud-controller-manager
ENTRYPOINT ["/bin/hcloud-cloud-controller-manager"]
