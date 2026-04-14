BINARY      := systemd-resolved-exporter
PKG         := github.com/konradasb/systemd-resolved-exporter
CMD         := ./cmd/$(BINARY)
BIN_DIR     := bin

VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
REVISION    ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BRANCH      ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
BUILD_USER  ?= $(shell whoami)
BUILD_DATE  ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

VERSION_PKG := github.com/prometheus/common/version
LDFLAGS := -s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Revision=$(REVISION) \
	-X $(VERSION_PKG).Branch=$(BRANCH) \
	-X $(VERSION_PKG).BuildUser=$(BUILD_USER) \
	-X $(VERSION_PKG).BuildDate=$(BUILD_DATE)

GO          ?= go
GOFLAGS     ?= -trimpath

.PHONY: all
all: tidy fmt vet lint test build

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:
	$(GO) test $(GOFLAGS) -race -covermode=atomic -coverprofile=coverage.out ./...

.PHONY: cover
cover: test
	$(GO) tool cover -func=coverage.out

.PHONY: build
build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) $(CMD)

.PHONY: run
run:
	$(GO) run $(CMD) --log.level=debug --collect-mode=cli

.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --clean

.PHONY: release
release:
	goreleaser release --clean

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) dist coverage.out
