SHELL := /bin/bash
LATEST_VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date +%s)
.DEFAULT_GOAL := build

INSTALL_DIR ?= $(HOME)/.local/bin
LDFLAGS_PKG = github.com/Surfe/surfer/pkg
LDFLAGS = -X $(LDFLAGS_PKG).BuildVersion=$(LATEST_VERSION) \
          -X $(LDFLAGS_PKG).BuildCommit=$(COMMIT) \
          -X $(LDFLAGS_PKG).BuildTime=$(BUILD_TIME)

GOLANGCI_LINT_RUN = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6 run
GORELEASER = go run github.com/goreleaser/goreleaser/v2@v2.5.1

build:
	go build -ldflags '$(LDFLAGS)' -o surfer

install: build
	@mkdir -p $(INSTALL_DIR)
	cp surfer $(INSTALL_DIR)/surfer
	@echo "Installed surfer to $(INSTALL_DIR)/surfer"

lint:
	$(GOLANGCI_LINT_RUN)

lint-fix:
	$(GOLANGCI_LINT_RUN) --fix

test:
	go test -coverpkg=./... -coverprofile c.out ./...

cover: test
	go tool cover -func c.out | tail -1

release-snapshot:
	$(GORELEASER) --snapshot --skip-publish --clean
