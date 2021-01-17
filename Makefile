SHELL := /bin/bash

TAG = dev
IMG ?= azad-kube-proxy:$(TAG)
TEST_ENV_FILE = tmp/test_env

ifneq (,$(wildcard $(TEST_ENV_FILE)))
    include $(TEST_ENV_FILE)
    export
endif

.SILENT:
lint:
	golangci-lint run

.SILENT:
fmt:
	go fmt ./...

.SILENT:
vet:
	go vet ./...

.SILENT:
test:
	go test -timeout 1m ./pkg/handlers -cover

.SILENT:
run:
	go run cmd/azad-kube-proxy/main.go --client-id="${CLIENT_ID}" --client-secret="${CLIENT_SECRET}" --tenant-id="${TENANT_ID}" --azure-ad-group-prefix="${AZURE_AD_GROUP_PREFIX}" --kubernetes-api-host="${KUBERNETES_API_HOST}" --kubernetes-api-port="${KUBERNETES_API_PORT}" --kubernetes-api-ca-cert-path="${KUBERNETES_API_CA_CERT_PATH}" --kubernetes-api-token-path="${KUBERNETES_API_TOKEN_PATH}" --tls-enabled="${TLS_ENABLED}" --tls-certificate-path="${TLS_CERTIFICATE_PATH}" --tls-key-path="${TLS_KEY_PATH}" --port="${PORT}"

.SILENT:
token:
	 az account get-access-token --resource ${TEST_USER_SP_RESOURCE} --query accessToken --output tsv

.SILENT:
docker-build:
	docker build -t $(IMG) .

.SILENT:
kind-load:
	kind load docker-image $(IMG)

.SILENT:
build:
	go build -o bin/azad-kube-proxy cmd/azad-kube-proxy/main.go
