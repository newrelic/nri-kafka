WORKDIR      := $(shell pwd)
TARGET       := target
TARGET_DIR    = $(WORKDIR)/$(TARGET)
NATIVEOS	 := $(shell go version | awk -F '[ /]' '{print $$4}')
NATIVEARCH	 := $(shell go version | awk -F '[ /]' '{print $$5}')
INTEGRATION  := kafka
BINARY_NAME   = nri-$(INTEGRATION)
GO_PKGS      := $(shell go list ./... | grep -v "/vendor/")
GO_FILES     := ./src/
GOTOOLS       = github.com/kardianos/govendor \
		github.com/axw/gocov/gocov \
		github.com/AlekSi/gocov-xml \

GOLANGCI_LINT_VERSION := v1.30.0
GOLANGCI_LINT_BIN = $(GOPATH)/bin/golangci-lint

all: build

build: check-version clean validate test compile

generate: tools
	@echo "=== $(INTEGRATION) === [ generate ]: Generating mocks..."
	@go generate ./src/connection/...

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin coverage.xml $(TARGET)

$(GOLANGCI_LINT_BIN):
	@echo "installing GolangCI version $(GOLANGCI_LINT_VERSION)"
	@(curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION))


tools: check-version $(GOLANGCI_LINT_BIN)
	@echo "=== $(INTEGRATION) === [ tools ]: Installing tools required by the project..."
	@go get $(GOTOOLS)

tools-update: check-version
	@echo "=== $(INTEGRATION) === [ tools-update ]: Updating tools required by the project..."
	@go get -u $(GOTOOLS)

deps: tools deps-only

deps-only:
	@echo "=== $(INTEGRATION) === [ deps ]: Installing package dependencies required by the project..."
	@govendor sync

validate: deps
	@echo "=== $(INTEGRATION) === [ validate ]: Validating source code ..."
	@$(GOLANGCI_LINT_BIN) run --timeout 90s

compile: deps
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./src

compile-only: deps-only
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./src

test: deps
	@echo "=== $(INTEGRATION) === [ test ]: Running unit tests..."
	@gocov test -race $(GO_PKGS) | gocov-xml > coverage.xml

# Include thematic Makefiles
include Makefile-*.mk

check-version:
ifdef GOOS
ifneq "$(GOOS)" "$(NATIVEOS)"
	$(error GOOS is not $(NATIVEOS). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif
ifdef GOARCH
ifneq "$(GOARCH)" "$(NATIVEARCH)"
	$(error GOARCH variable is not $(NATIVEARCH). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif

.PHONY: all build clean tools tools-update deps validate compile test check-version
