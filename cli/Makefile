# HiddenLayer APIM CLI Makefile

GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BASE_VERSION ?= 0.1.0
DIRTY_SUFFIX ?= $(shell test -n "$$(git status --porcelain 2>/dev/null)" && echo ".dirty")
VERSION ?= $(shell \
	EXACT_TAG="$$(git describe --tags --exact-match 2>/dev/null)"; \
	if echo "$$EXACT_TAG" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$$'; then \
		echo "$$EXACT_TAG"; \
	else \
		echo "v$(BASE_VERSION)-dev+$(GIT_COMMIT)$(DIRTY_SUFFIX)"; \
	fi \
)

LDFLAGS := -ldflags "-X github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/cmd.Version=$(VERSION) \
	-X github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/cmd.GitCommit=$(GIT_COMMIT) \
	-X github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/cmd.BuildDate=$(BUILD_DATE)"

.PHONY: build clean install test lint all

all: build

build:
	go build $(LDFLAGS) -o hiddenlayer-apim .

install:
	go install $(LDFLAGS) .

clean:
	rm -f hiddenlayer-apim
	rm -rf dist/

test:
	go test -v ./...

lint:
	golangci-lint run

# Cross-compilation targets
.PHONY: build-all build-linux build-darwin build-windows

build-all: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/hiddenlayer-apim-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/hiddenlayer-apim-linux-arm64 .

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/hiddenlayer-apim-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/hiddenlayer-apim-darwin-arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/hiddenlayer-apim-windows-amd64.exe .

# Release target (creates checksums)
.PHONY: release
release: build-all
	cd dist && sha256sum * > checksums.txt
