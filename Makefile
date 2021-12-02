SHELL := /bin/bash

TAG = dev
IMG ?= azad-kube-proxy:$(TAG)
TEST_ENV_FILE = tmp/test_env
VERSION ?= "v0.0.0-dev"
REVISION ?= ""
CREATED ?= ""

ifneq (,$(wildcard $(TEST_ENV_FILE)))
    include $(TEST_ENV_FILE)
    export
endif

.PHONY: all
.SILENT: all
all: tidy lint fmt vet gosec test build build-plugin

.PHONY: lint
.SILENT: lint
lint:
	golangci-lint run

.PHONY: fmt
.SILENT: fmt
fmt:
	go fmt ./...

.PHONY: tidy
.SILENT: tidy
tidy:
	go mod tidy

.PHONY: vet
.SILENT: vet
vet:
	go vet ./...

.PHONY: test 
.SILENT: test
test:
	mkdir -p tmp
	go test -timeout 1m ./... -cover

.PHONY: gosec
.SILENT: gosec
gosec:
	gosec ./...

.PHONY: cover
.SILENT: cover
cover:
	go test -timeout 1m ./... -coverprofile=tmp/coverage.out
	go tool cover -html=tmp/coverage.out	

.PHONY: run
.SILENT: run
run:
	go run cmd/azad-kube-proxy/main.go --client-id="${CLIENT_ID}" --client-secret="${CLIENT_SECRET}" --tenant-id="${TENANT_ID}" --azure-ad-group-prefix="${AZURE_AD_GROUP_PREFIX}" --kubernetes-api-host="${KUBERNETES_API_HOST}" --kubernetes-api-port="${KUBERNETES_API_PORT}" --kubernetes-api-ca-cert-path="${KUBERNETES_API_CA_CERT_PATH}" --kubernetes-api-token-path="${KUBERNETES_API_TOKEN_PATH}" --tls-enabled="${TLS_ENABLED}" --tls-certificate-path="${TLS_CERTIFICATE_PATH}" --tls-key-path="${TLS_KEY_PATH}" --port="${PORT}"

.PHONY: run-plugin
.SILENT: run-plugin
run-plugin:
	go run cmd/kubectl-azad-proxy/main.go

.PHONY: debug
.SILENT: debug
debug:
	dlv debug cmd/azad-kube-proxy/main.go --listen=:40000 --headless=true --api-version=2 --log -- --client-id="${CLIENT_ID}" --client-secret="${CLIENT_SECRET}" --tenant-id="${TENANT_ID}" --azure-ad-group-prefix="${AZURE_AD_GROUP_PREFIX}" --kubernetes-api-host="${KUBERNETES_API_HOST}" --kubernetes-api-port="${KUBERNETES_API_PORT}" --kubernetes-api-ca-cert-path="${KUBERNETES_API_CA_CERT_PATH}" --kubernetes-api-token-path="${KUBERNETES_API_TOKEN_PATH}" --tls-enabled="${TLS_ENABLED}" --tls-certificate-path="${TLS_CERTIFICATE_PATH}" --tls-key-path="${TLS_KEY_PATH}" --port="${PORT}"

.PHONY: token
.SILENT: token
token:
	 az account get-access-token --resource ${TEST_USER_SP_RESOURCE} --query accessToken --output tsv

.PHONY: build
.SILENT: build
build:
	CGO_ENABLED=0 go build -ldflags "-w -s -X main.Version=$(VERSION) -X main.Revision=$(REVISION) -X main.Created=$(CREATED)" -o bin/azad-kube-proxy cmd/azad-kube-proxy/main.go

.PHONY: build-plugin
.SILENT: build-plugin
build-plugin:
	go build -o bin/kubectl-azad_proxy cmd/kubectl-azad-proxy/main.go
