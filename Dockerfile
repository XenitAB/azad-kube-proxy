# BUILDER
FROM golang:1.15-alpine as builder
WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY Makefile Makefile
COPY cmd/ cmd/
COPY pkg/ pkg/

RUN apk add --no-cache make bash
RUN make build

#RUNTIME
FROM alpine:3.12 as runtime
LABEL org.opencontainers.image.source="https://github.com/XenitAB/azad-kube-proxy"
RUN apk add --no-cache ca-certificates tini

WORKDIR /
COPY --from=builder /workspace/bin/azad-kube-proxy /usr/local/bin/

RUN [ ! -e /etc/nsswitch.conf ] && echo "hosts: files dns" > /etc/nsswitch.conf

RUN addgroup -S azad-kube-proxy && adduser -S -g azad-kube-proxy azad-kube-proxy
USER azad-kube-proxy

ENTRYPOINT [ "/sbin/tini", "--", "azad-kube-proxy"]