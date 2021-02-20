SHELL := /bin/bash

TAG = dev
IMG ?= azad-kube-proxy:$(TAG)
TEST_ENV_FILE = tmp/test_env
VERSION ?= "v0.0.0-dev"
REVISION ?= ""
CREATED ?= ""
K8DASH_DIR ?= ${PWD}/pkg/dashboard/static/k8dash


ifneq (,$(wildcard $(TEST_ENV_FILE)))
    include $(TEST_ENV_FILE)
    export
endif

.SILENT:
all: tidy lint fmt vet gosec go-test build build-plugin

.SILENT:
lint:
	golangci-lint run

.SILENT:
fmt:
	go fmt ./...

.SILENT:
tidy:
	go mod tidy

.SILENT:
vet:
	go vet ./...

.SILENT:
go-test:
	mkdir -p tmp
	go test -timeout 1m ./... -cover

.SILENT:
gosec:
	gosec ./...

.SILENT:
cover:
	go test -timeout 1m ./... -coverprofile=tmp/coverage.out                                                                                                                                                                                         16:10:38
	go tool cover -html=tmp/coverage.out	

.SILENT:
run:
	go run cmd/azad-kube-proxy/main.go --client-id="${CLIENT_ID}" --client-secret="${CLIENT_SECRET}" --tenant-id="${TENANT_ID}" --azure-ad-group-prefix="${AZURE_AD_GROUP_PREFIX}" --kubernetes-api-host="${KUBERNETES_API_HOST}" --kubernetes-api-port="${KUBERNETES_API_PORT}" --kubernetes-api-ca-cert-path="${KUBERNETES_API_CA_CERT_PATH}" --kubernetes-api-token-path="${KUBERNETES_API_TOKEN_PATH}" --tls-enabled="${TLS_ENABLED}" --tls-certificate-path="${TLS_CERTIFICATE_PATH}" --tls-key-path="${TLS_KEY_PATH}" --port="${PORT}"

.SILENT:
run-plugin:
	go run cmd/kubectl-azad-proxy/main.go

.SILENT:
debug:
	dlv debug cmd/azad-kube-proxy/main.go --listen=:40000 --headless=true --api-version=2 --log -- --client-id="${CLIENT_ID}" --client-secret="${CLIENT_SECRET}" --tenant-id="${TENANT_ID}" --azure-ad-group-prefix="${AZURE_AD_GROUP_PREFIX}" --kubernetes-api-host="${KUBERNETES_API_HOST}" --kubernetes-api-port="${KUBERNETES_API_PORT}" --kubernetes-api-ca-cert-path="${KUBERNETES_API_CA_CERT_PATH}" --kubernetes-api-token-path="${KUBERNETES_API_TOKEN_PATH}" --tls-enabled="${TLS_ENABLED}" --tls-certificate-path="${TLS_CERTIFICATE_PATH}" --tls-key-path="${TLS_KEY_PATH}" --port="${PORT}"

.SILENT:
token:
	 az account get-access-token --resource ${TEST_USER_SP_RESOURCE} --query accessToken --output tsv

.SILENT:
build:
	go build -ldflags "-w -s -X main.Version=$(VERSION) -X main.Revision=$(REVISION) -X main.Created=$(CREATED)" -o bin/azad-kube-proxy cmd/azad-kube-proxy/main.go

.SILENT:
build-plugin:
	go build -o bin/kubectl-azad_proxy cmd/kubectl-azad-proxy/main.go

.SILENT:
build-k8dash:
	git submodule init
	git submodule update
	docker build gitmodules/k8dash -t k8dash:build-deps --target build-deps
	rm -rf $(K8DASH_DIR)
	mkdir -p $(K8DASH_DIR)
	docker create --name k8dash-build-deps k8dash:build-deps
	docker cp k8dash-build-deps:/usr/src/app/build/ $(K8DASH_DIR)
	docker rm k8dash-build-deps
	cp gitmodules/k8dash/LICENSE $(K8DASH_DIR)/
