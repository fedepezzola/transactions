# Makefile for Go projects.
#
# This Makefile makes an effort to provide standard make targets, as described
# by https://www.gnu.org/prep/standards/html_node/Standard-Targets.html.

include Makefile.*
SHELL := /bin/sh

GOIMPORTS := $(GORUN) golang.org/x/tools/cmd/goimports@v0.1.11
GOLANGCI_LINT := $(GORUN) github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2
GOFMT := $(GORUN) mvdan.cc/gofumpt@v0.3.1
GOSWAG := $(GORUN) github.com/swaggo/swag/cmd/swag@v1.8.10
GIT := git
MIGRATE := migrate
COVERAGE_FILE := coverage.out
COVER_FILTER_MOCKS := | grep -vi mock_ | grep -vi generated_
DEST_BIN := ./dist

BUILD_OUTPUT := -o $(DEST_BIN)
BUILD_DEBUG_FLAGS := -gcflags "-N -l"
SOURCES := $(shell \
	find . -not \( \( -name .git -o -name .go -o -name vendor \) -prune \) \
	-name '*.go')
NON_GENERATED_SOURCES := $(shell \
	find . -not \( \( -name .git -o -name .go -o -name vendor \) -prune \) \
	-name '*.go' -not -name 'mock_*.go' -not -name 'generated_*.go')
GENERATED_SOURCES := $(shell \
	find . -not \( \( -name .git -o -name .go -o -name vendor \) -prune \) \
	-name 'mock_*.go' -o -name 'generated_*.go')
ifdef DEBUG
$(info SOURCES = $(SOURCES))
endif

################################################################################
## Version information
################################################################################

VERSION := $(shell $(GIT) describe --tags --always --dirty 2> /dev/null || echo "unknown-by-$(shell whoami)" | echo "unknown")
GIT_HEAD := $(shell $(GIT) rev-parse --verify HEAD)$(shell $(GIT) diff --quiet || echo '-dirty')
BUILD := git-$(GIT_HEAD)@$(shell date +%Y-%m-%dT%H:%M:%S%z)
BUILD_INFO_LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.build=$(BUILD)"

version:
	@echo "version: $(VERSION). githead: $(GIT_HEAD). build: $(BUILD)"

################################################################################
## Standard make targets
################################################################################

.DEFAULT_GOAL := all
.PHONY: all
all: fix lint build

.PHONY: install
install:
	$(GOINSTALL) ./

# TODO(alan.parra): Update uninstall target
.PHONY: uninstall
uninstall:
	$(RM) $$GOPATH/bin/app

.PHONY: clean
clean:
	$(RM) $(COVERAGE_FILE) coverage.xml
	$(RM) $(GENERATED_SOURCES)

.PHONY: check
check: test

.PHONY: deps
deps:
	$(GO) install \
	    github.com/t-yuki/gocover-cobertura \
	    golang.org/x/tools/cmd/goimports \
	    github.com/topfreegames/helm-generate/cmd/helm-generate \


.PHONY: run
run:
	$(GO) run $(BUILD_INFO_LDFLAGS) transactions.go

.PHONY: generate
generate:
	$(GO) generate ./...


.PHONY: create-database
create-database:
	docker-compose exec db psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$(TRANSACTIONS_DB_NAME)';" | grep 1 -q || \
	docker-compose exec db psql -U postgres -tc "CREATE DATABASE $(TRANSACTIONS_DB_NAME)"

.PHONY: drop-database
drop-database:
	docker-compose exec db psql -U postgres -tc "DROP DATABASE $(TRANSACTIONS_DB_NAME);"

.PHONY: migrate
migrate:
	echo $(PWD)
	$(MIGRATE) -source file://infrastructure/database/migrations \
			-database "postgres://$(TRANSACTIONS_DB_USER):$(TRANSACTIONS_DB_PASSWORD)@$(DB_HOST):5432/$(TRANSACTIONS_DB_NAME)?sslmode=disable" up

.PHONY: create-migration
create-migration:
	$(MIGRATE) create -ext sql -dir infrastructure/database/migrations/ -seq $(ARGS)

################################################################################
## Go-like targets
################################################################################

.PHONY: build
build: generate build-only

.PHONY: build-only
build-only:
	mkdir -p $(DEST_BIN)
	$(GOBUILD) $(BUILD_INFO_LDFLAGS) $(BUILD_OUTPUT) -buildvcs=false ./...

.PHONY: build-debug
build-debug:
	mkdir -p $(DEST_BIN)
	$(GOBUILD) $(BUILD_INFO_LDFLAGS) $(BUILD_OUTPUT) $(BUILD_DEBUG_FLAGS) -buildvcs=false ./...

.PHONY: test
test:
	$(GOTEST) -coverprofile=$(COVERAGE_FILE).tmp -covermode=count ./... $(SILENT_CMD_SUFFIX)
	cat $(COVERAGE_FILE).tmp $(COVER_FILTER_MOCKS) > $(COVERAGE_FILE)
	$(RM) $(COVERAGE_FILE).tmp

.PHONY: test-package
test-package:
	$(GOTEST) $(PACKAGE)

.PHONY: cover
cover: cover/text

.PHONY: cover/html
cover/html: test
	$(GOTOOL) cover -html=$(COVERAGE_FILE)

.PHONY: cover/text
cover/text: test
	$(GOTOOL) cover -func=$(COVERAGE_FILE)

################################################################################
## Linters and formatters
################################################################################

.PHONY: fix
fix:
ifeq (,$(RUNNING_IN_CI))
	$(GOMOD) tidy
endif
ifneq ($(NON_GENERATED_SOURCES),)
	$(GOIMPORTS) -w $(NON_GENERATED_SOURCES)
	$(GOFMT) -w $(NON_GENERATED_SOURCES)
endif

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run --timeout=5m

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix

.PHONY: manifests
manifests:
	helm-generate ./k8s

.PHONY: ci-fmt
ci-fmt:
	@test -z "$$($(GOFMT) -d $(NON_GENERATED_SOURCES))" || (printf "Formatting check failed!\n\nPlease fix it with 'make fix'\n" && false)

