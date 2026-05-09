GO ?= go
BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/openstack
GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/gomod

export GOCACHE
export GOMODCACHE

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show available Make targets.
	@awk 'BEGIN {FS = ":.*##"; printf "Available targets:\n"} /^[a-zA-Z0-9_.-]+:.*##/ { printf "  %-24s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf "\nLifecycle suites for 'make lifecycle CLOUD=name SUITE=suite':\n"
	@printf "  %-24s %s\n" "keypair" "Create, show, and delete a disposable keypair."
	@printf "  %-24s %s\n" "server" "Run disposable Compute server lifecycle and attach/detach checks."
	@printf "  %-24s %s\n" "volume" "Run disposable Block Storage volume lifecycle checks."
	@printf "  %-24s %s\n" "quota" "Run project-scoped quota mutation checks."
	@printf "  %-24s %s\n" "image" "Run disposable Image service lifecycle and metadef checks."
	@printf "  %-24s %s\n" "network" "Run disposable Network service lifecycle checks."
	@printf "  %-24s %s\n" "object" "Run disposable Object Storage container and object checks."
	@printf "  %-24s %s\n" "all" "Run all lifecycle suites in sequence."

.PHONY: build
build: ## Build the drop-in openstack binary at bin/openstack.
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) ./cmd/openstack

.PHONY: test
test: ## Run the full Go test suite.
	$(GO) test ./...

.PHONY: fmt
fmt: ## Format all Go source files.
	$(GO) fmt ./...

.PHONY: check
check: fmt test build ## Format, test, and build the CLI.

.PHONY: smoke
smoke: build ## Run basic local CLI smoke checks that do not require cloud access.
	$(BINARY) --version
	$(BINARY) command list -f json --group openstack.cli
	$(BINARY) server list --help
	$(BINARY) module list --max-width 52

.PHONY: catalog
catalog: ## Regenerate Python OSC compatibility catalog artifacts.
	$(GO) run ./tools/osc-catalog

.PHONY: matrix
matrix: ## Regenerate compatibility and test matrix artifacts.
	$(GO) run ./tools/matrix

.PHONY: report
report: ## Print README-ready command compatibility Markdown table.
	$(GO) run ./tools/matrix --report command-status --report-format readme

.PHONY: discover
discover: ## Discover non-secret live cloud capabilities; set CLOUD=name[,name].
	$(GO) run ./tools/cloud-discovery --cloud "$(CLOUD)"

.PHONY: lifecycle
lifecycle: ## Run lifecycle tests; set CLOUD=name and optionally SUITE=name.
	$(GO) run ./tools/lifecycle-smoke --cloud "$(CLOUD)" --suite "$(or $(SUITE),keypair)"

.PHONY: os-test
os-test: ## Display supported Fancy operating-system image colors.
	$(GO) run ./tools/os-test

.PHONY: compat
compat: catalog matrix ## Regenerate all compatibility artifacts.

.PHONY: clean
clean: ## Remove local build outputs and workspace-local Go caches.
	$(RM) -r $(BIN_DIR) .cache

.PHONY: mrproper
mrproper: clean ## Remove local build outputs and workspace-local Go caches.
	$(RM) -r .cache
