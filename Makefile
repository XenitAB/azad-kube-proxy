SHELL:=/bin/bash

TAG = dev
IMG ?= azad-kube-proxy:$(TAG)
TEST_ENV_FILE = tmp/env

ifneq (,$(wildcard $(TEST_ENV_FILE)))
    include $(TEST_ENV_FILE)
    export
endif

lint:
	golangci-lint run

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	echo $(TENANT_ID)
	go test -timeout 1m ./... -cover -v

docker-build:
	docker build -t $(IMG) .

kind-load:
	kind load docker-image $(IMG)

build:
	go build -o bin/azad-kube-proxy cmd/azad-kube-proxy/main.go
