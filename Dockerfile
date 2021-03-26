# BUILDER
FROM golang:1.16-alpine as builder

ARG VERSION
ARG REVISION
ARG CREATED

ENV VERSION=$VERSION
ENV REVISION=$REVISION
ENV CREATED=$CREATED

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY Makefile Makefile
COPY cmd/ cmd/
COPY pkg/ pkg/

RUN apk add --no-cache make=4.3-r0 bash=5.1.0-r0
RUN make build

#RUNTIME
FROM alpine:3.13.3 as runtime
LABEL org.opencontainers.image.source="https://github.com/XenitAB/azad-kube-proxy"

# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates

RUN apk add --no-cache tini=0.19.0-r0

WORKDIR /
COPY --from=builder /workspace/bin/azad-kube-proxy /usr/local/bin/

RUN [ ! -e /etc/nsswitch.conf ] && echo "hosts: files dns" > /etc/nsswitch.conf

RUN addgroup -S azad-kube-proxy && adduser -S -g azad-kube-proxy azad-kube-proxy
USER azad-kube-proxy

ENTRYPOINT [ "/sbin/tini", "--", "azad-kube-proxy"]