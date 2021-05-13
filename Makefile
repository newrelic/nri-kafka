WORKDIR      := $(shell pwd)
TARGET       := target
TARGET_DIR    = $(WORKDIR)/$(TARGET)
NATIVEOS	 := $(shell go version | awk -F '[ /]' '{print $$4}')
NATIVEARCH	 := $(shell go version | awk -F '[ /]' '{print $$5}')
INTEGRATION  := kafka
BINARY_NAME   = nri-$(INTEGRATION)
GO_PKGS      := $(shell go list ./... | grep -v "/vendor/")
GO_FILES     := ./src/
GOFLAGS          = -mod=readonly
GOLANGCI_LINT    = github.com/golangci/golangci-lint/cmd/golangci-lint
GOCOV            = github.com/axw/gocov/gocov
GOCOV_XML        = github.com/AlekSi/gocov-xml

all: build

build: clean validate test compile

generate:
	@echo "=== $(INTEGRATION) === [ generate ]: Generating mocks..."
	@go generate ./src/connection/...

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin coverage.xml $(TARGET)

validate:
ifeq ($(strip $(GO_FILES)),)
	@echo "=== $(INTEGRATION) === [ validate ]: no Go files found. Skipping validation."
else
	@printf "=== $(INTEGRATION) === [ validate ]: running golangci-lint & semgrep... "
	@go run  $(GOFLAGS) $(GOLANGCI_LINT) run --verbose
	@if [ -f .semgrep.yml ]; then \
        docker run --rm -v "${PWD}:/src:ro" --workdir /src returntocorp/semgrep -c .semgrep.yml ; \
    else \
    	docker run --rm -v "${PWD}:/src:ro" --workdir /src returntocorp/semgrep -c p/golang ; \
    fi
endif

compile:
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./src

test:
	@echo "=== $(INTEGRATION) === [ test ]: running unit tests..."
	@go run $(GOFLAGS) $(GOCOV) test ./... | go run $(GOFLAGS) $(GOCOV_XML) > coverage.xml

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

.PHONY: all build clean validate compile test
