WORKDIR      := $(shell pwd)
TARGET       := target
TARGET_DIR    = $(WORKDIR)/$(TARGET)
NATIVEOS	 := $(shell go version | awk -F '[ /]' '{print $$4}')
NATIVEARCH	 := $(shell go version | awk -F '[ /]' '{print $$5}')
INTEGRATION  := kafka
BINARY_NAME   = nri-$(INTEGRATION)
GO_PKGS      := $(shell go list ./... | grep -v "/vendor/")
GO_FILES     := ./src/
GOFLAGS       = -mod=readonly
GO_VERSION 		?= $(shell grep '^go ' go.mod | awk '{print $$2}')
BUILDER_IMAGE 	?= "ghcr.io/newrelic/coreint-automation:latest-go$(GO_VERSION)-ubuntu16.04"

all: build

build: clean test compile

generate:
	@echo "=== $(INTEGRATION) === [ generate ]: Generating mocks..."
	@go generate ./src/connection/...

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin coverage.xml $(TARGET)

compile:
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./src

test:
	@echo "=== $(INTEGRATION) === [ test ]: running unit tests..."
	@go test -race ./... -count=1

integration-test:
	@echo "=== $(INTEGRATION) === [ test ]: running integration tests..."
	@if [ "$(NRJMX_VERSION)" = "" ]; then \
	    echo "Error: missing required env-var: NRJMX_VERSION\n" ;\
        exit 1 ;\
	fi
	@cp tests/integration/jmxremote/jmxremote.password.unencoded tests/integration/jmxremote/jmxremote.password
	@chmod 0600 tests/integration/jmxremote/jmxremote.password
	@docker compose -f tests/integration/docker-compose.yml up -d --build
	@go test -v -tags=integration ./tests/integration/. -count=1 ; (ret=$$?; docker compose -f tests/integration/docker-compose.yml down && exit $$ret)

POD_NAME  := agent
NAMESPACE := test-kafka
ARGS := ""
# run an agent pod with "kubectl run   agent --image newrelic/infrastructure-bundle --env="LICENSE_KEY=...."
run-on-pod:
	GOOS=linux GOARCH=amd64 make compile
	kubectl cp ./bin/nri-kafka $(POD_NAME):/nri-kafka -n $(NAMESPACE)
	@echo "=== $(INTEGRATION) === [ test ]: running kafka on pod..."
	@echo "set ARGS to be passed to kafka binary"
	kubectl exec -n $(NAMESPACE) $(POD_NAME) -- /nri-kafka  $(ARGS)

# rt-update-changelog runs the release-toolkit run.sh script by piping it into bash to update the CHANGELOG.md.
# It also passes down to the script all the flags added to the make target. To check all the accepted flags,
# see: https://github.com/newrelic/release-toolkit/blob/main/contrib/ohi-release-notes/run.sh
#  e.g. `make rt-update-changelog -- -v`
rt-update-changelog:
	curl "https://raw.githubusercontent.com/newrelic/release-toolkit/v1/contrib/ohi-release-notes/run.sh" | bash -s -- $(filter-out $@,$(MAKECMDGOALS))

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

.PHONY: all build clean compile test rt-update-changelog
