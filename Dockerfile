FROM golang:1.14-alpine3.11 as builder
ADD *.go /proxy/
WORKDIR /proxy
RUN go build -o /tmp/proxy

FROM alpine:3.11
COPY --from=builder /tmp/proxy /usr/bin/stupid-proxy

ENTRYPOINT [ "stupid-proxy" ]
