# Set V to 1 for verbose output from the Makefile
Q=$(if $V,,@)
SRC=$(shell find . -type f -name '*.go')

all: lint test

ci: test

.PHONY: all ci

#########################################
# Bootstrapping
#########################################

bootstra%:
	$Q curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest
	$Q go install golang.org/x/vuln/cmd/govulncheck@latest
	$Q go install gotest.tools/gotestsum@latest

.PHONY: bootstrap

#########################################
# Test
#########################################

test:
	go test -cover ./...

# don't run race tests by default. see https://github.com/etcd-io/bbolt/issues/187
test-race:
	go test -cover -race ./...

.PHONY: test test-race

#########################################
# Linting
#########################################

fmt:
	$Q goimports -l -w $(SRC)

lint: golint govulncheck

golint: SHELL:=/bin/bash
golint:
	$Q LOG_LEVEL=error golangci-lint run --config <(curl -s https://raw.githubusercontent.com/smallstep/workflows/master/.golangci.yml) --timeout=30m

govulncheck:
	$Q govulncheck ./...

.PHONY: fmt lint golint govulncheck
