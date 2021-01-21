FROM golang:1.15 as builder
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY Makefile Makefile
COPY cmd/ cmd/
COPY pkg/ pkg/
RUN make build

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/bin/azad-kube-proxy .
USER nonroot:nonroot
ENTRYPOINT ["/azad-kube-proxy"]