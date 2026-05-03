# Makefile for harchos-terraform-provider

NAME = terraform-provider-harchos
BINARY = $(NAME)
VERSION ?= 0.1.0

# Go build variables
GO ?= go
GOFLAGS ?= -trimpath -ldflags="-s -w -X main.version=$(VERSION)"
CMD_PATH = .

# Build directory
BUILD_DIR = ./bin

.PHONY: all build install test testacc clean fmt vet lint generate docs

all: build

build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD_PATH)

install: build
	$(GO) install $(GOFLAGS) $(CMD_PATH)

test:
	$(GO) test -v -count=1 -timeout 120s ./...

testacc:
	TF_ACC=1 $(GO) test -v -count=1 -timeout 120s ./...

clean:
	rm -rf $(BUILD_DIR)
	$(GO) clean

fmt:
	gofmt -w .
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	golangci-lint run ./...

generate:
	$(GO) generate ./...

docs:
	$(GO) run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Development install for local testing
dev: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/harchcorp/harchos/$(VERSION)/$(shell go env GOOS)_$(shell go env GOARCH)
	cp $(BUILD_DIR)/$(BINARY) ~/.terraform.d/plugins/registry.terraform.io/harchcorp/harchos/$(VERSION)/$(shell go env GOOS)_$(shell go env GOARCH)/

# Run acceptance tests with verbose output
testacc-verbose:
	TF_ACC=1 $(GO) test -v -count=1 -timeout 300s -run TestAcc ./...

# Check for common issues
check: fmt vet lint test
	@echo "All checks passed!"
