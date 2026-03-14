# Set V to 1 for verbose output from the Makefile
Q=$(if $V,,@)
SRC=$(shell find . -type f -name '*.go')

# protoc-gen-go constraints
GEN_GO_BIN ?= protoc-gen-go
GEN_GO_MIN_VERSION ?= 1.36.1
GEN_GO_VERSION ?= $(shell $(GEN_GO_BIN) --version | awk -F ' v' '{print $$NF}')

# protoc-gen-go-grpc constraints
GEN_GRPC_BIN ?= protoc-gen-go-grpc
GEN_GRPC_MIN_VERSION ?= 1.5.1
GEN_GRPC_VERSION ?= $(shell $(GEN_GRPC_BIN) --version | awk -F ' ' '{print $$NF}')

# Go tools
GOIMPORTS=golang.org/x/tools/cmd/goimports
GOLANGCI_LINT=github.com/golangci/golangci-lint/v2/cmd/golangci-lint
GOTESTSUM=gotest.tools/gotestsum
GOVULNCHECK=golang.org/x/vuln/cmd/govulncheck

all: lint generate test

ci: test

.PHONY: all ci

#########################################
# Build
#########################################

build: ;

#########################################
# Bootstrapping
#########################################

bootstra%:
	$Q go install -mod=readonly google.golang.org/protobuf/cmd/protoc-gen-go
	$Q go install -mod=readonly google.golang.org/grpc/cmd/protoc-gen-go-grpc

.PHONY: bootstrap

#########################################
# Test
#########################################

test:
	$Q $(GOFLAGS) go tool $(GOTESTSUM) -- -coverpkg=./... -coverprofile=coverage.out -covermode=atomic ./...

race:
	$Q $(GOFLAGS) go tool $(GOTESTSUM) -- -race ./...

.PHONY: test race

#########################################
# Linting
#########################################

fmt:
	$Q go tool $(GOIMPORTS) -local github.com/smallstep/linkedca -l -w $(SRC)

lint: SHELL:=/bin/bash
lint:
	$Q LOG_LEVEL=error go tool $(GOLANGCI_LINT) run --config <(curl -s https://raw.githubusercontent.com/smallstep/workflows/master/.golangci.yml) --timeout=30m
	$Q go tool $(GOVULNCHECK) ./...

.PHONY: fmt lint

#########################################
# Generate
#########################################

generate: check-gen-go-version check-gen-grpc-version
	@# remove any previously generated protobufs & gRPC files
	@find . \
		-type f \
		-name "*.pb.go" \
		-delete

	@# generate the corresponding protobufs & gRPC code files
	$Q protoc \
		--proto_path=spec \
		--go_opt=module=github.com/smallstep/linkedca \
		--go_out=. \
		--go-grpc_opt=module=github.com/smallstep/linkedca \
		--go-grpc_out=. \
		$(shell find spec -type f -name "*.proto")	

.PHONY: generate

#########################################
# Tool constraints
#########################################

check-gen-go-version:
	@if ! printf "%s\n%s" "$(GEN_GO_MIN_VERSION)" "$(GEN_GO_VERSION)" | sort -V -C; then \
		echo "Your $(GEN_GO_BIN) version (v$(GEN_GO_VERSION)) is older than the minimum required (v$(GEN_GO_MIN_VERSION))."; \
		exit 1; \
	fi

.PHONY: check-gen-go-version

check-gen-grpc-version:
	@if ! printf "%s\n%s" "$(GEN_GRPC_MIN_VERSION)" "$(GEN_GRPC_VERSION)" | sort -V -C; then \
		echo "Your $(GEN_GRPC_BIN) version (v$(GEN_GRPC_VERSION)) is older than the minimum required (v$(GEN_GRPC_MIN_VERSION))."; \
		exit 1; \
	fi

.PHONY: check-gen-grpc-version
