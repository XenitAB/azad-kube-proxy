TAG = dev
IMG ?= azad-kube-proxy:$(TAG)

ifeq (run,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(RUN_ARGS):;@:)
endif

lint:
	golangci-lint run

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test -timeout 1m ./...

docker-build:
	docker build -t $(IMG) .

kind-load:
	kind load docker-image $(IMG)

build:
	go build -o bin/azad-kube-proxy cmd/azad-kube-proxy/main.go
