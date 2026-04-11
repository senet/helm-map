BINARY_NAME := helm-map
MODULE := github.com/senet/helm-map
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test lint clean install-local release

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/helm-map/

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

install-local: build
	@if [ -z "$$HELM_PLUGINS" ]; then \
		echo "HELM_PLUGINS is not set. Run 'helm env' to determine plugin directory."; \
		exit 1; \
	fi
	mkdir -p "$$HELM_PLUGINS/helm-map/bin"
	cp bin/$(BINARY_NAME) "$$HELM_PLUGINS/helm-map/bin/"
	cp plugin.yaml "$$HELM_PLUGINS/helm-map/"
	@echo "Installed to $$HELM_PLUGINS/helm-map/"

release:
	goreleaser release --clean

cover:
	go test -coverprofile=coverage.out ./internal/engine/...
	go tool cover -func=coverage.out
	@rm -f coverage.out
