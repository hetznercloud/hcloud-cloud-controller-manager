FROM golang:1.12 as builder
WORKDIR /maschine-controller/src
ADD . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o hcloud-maschine-controller.bin  .


FROM alpine:3.9
RUN apk add --no-cache ca-certificates bash
COPY --from=builder /maschine-controller/src/hcloud-maschine-controller.bin /bin/hcloud-cloud-controller-manager
ENTRYPOINT ["/bin/hcloud-cloud-controller-manager"]
