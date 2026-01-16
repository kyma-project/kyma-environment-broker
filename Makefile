GOLINT_VER = v2.8.0
ifeq (,$(GOLINT_TIMEOUT))
GOLINT_TIMEOUT=2m
endif

ifndef ARTIFACTS
	ARTIFACTS = ./bin
endif

ifndef GIT_SHA
	GIT_SHA = ${shell git describe --tags --always}
endif

 ## The headers are represented by '##@' like 'General' and the descriptions of given command is text after '##''.
.PHONY: help
help: 
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ General

.PHONY: verify
verify: test checks go-lint ## verify simulates same behaviour as 'verify' GitHub Action which run on every PR

.PHONY: checks
checks: check-go-mod-tidy ## run different Go related checks

.PHONY: go-lint
go-lint: go-lint-install ## linter config in file at root of project -> '.golangci.yaml'
	golangci-lint run --timeout=$(GOLINT_TIMEOUT)

go-lint-install: ## linter config in file at root of project -> '.golangci.yaml'
	@if [ "$(shell command golangci-lint version --short 2>/dev/null)" != "$(GOLINT_VER)" ]; then \
  		echo golangci in version $(GOLINT_VER) not found. will be downloaded; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLINT_VER); \
		echo golangci installed with version: $(shell command golangci-lint version --short 2>/dev/null); \
	fi;
	
##@ Tests

.PHONY: test 
test: ## run Go tests
	GODEBUG=fips140=only,tlsmlkem=0 GOFIPS140=v1.0.0 go test ./...

##@ Go checks 

.PHONY: check-go-mod-tidy
check-go-mod-tidy: ## check if go mod tidy needed
	go mod tidy -v
	@if [ -n "$$(git status -s go.*)" ]; then \
		echo -e "${RED}✗ go mod tidy modified go.mod or go.sum files${NC}"; \
		git status -s go.*; \
		exit 1; \
	fi;

##@ Development support commands

.PHONY: fix
fix: go-lint-install ## automatically fix code issues with formatting, imports, and linting
	@echo "Running automatic code fixes..."
	@echo "├── Running go mod tidy..."
	go mod tidy -v
	@echo "├── Running goimports (organize imports and format)..."
	go run golang.org/x/tools/cmd/goimports@latest -w -local github.com/kyma-project/kyma-environment-broker .
	@echo "├── Running gofmt (standard formatting)..."
	gofmt -s -w .
	@echo "├── Running golangci-lint auto-fixes..."
	golangci-lint run --fix
	@echo "All automatic fixes completed!"

.PHONY: format
format: ## format Go code using goimports and gofmt
	@echo "Formatting Go code..."
	go run golang.org/x/tools/cmd/goimports@latest -w -local github.com/kyma-project/kyma-environment-broker .
	gofmt -s -w .
	@echo "Code formatting completed!"

.PHONY: fix-lint-issues
fix-lint-issues: go-lint-install ## fix specific linter issues that can be auto-corrected
	@echo "Fixing auto-correctable linter issues..."
	@echo "├── Fixing unconvert issues..."
	go run github.com/mdempsky/unconvert@latest -apply ./...
	@echo "├── Running golangci-lint auto-fixes (gofmt, goimports, etc.)..."  
	golangci-lint run --fix || true
	@echo "Auto-correctable linter fixes completed!"
	@echo "Run 'make go-lint' to see remaining issues that require manual fixes"


##@ Tools

.PHONY: build-hap
build-hap:
	cd cmd/parser; go build -ldflags "-X main.gitCommit=$(GIT_SHA)" -o ../../$(ARTIFACTS)/hap

##@ Installation

.PHONY: install
install:
	./scripts/installation.sh $(VERSION) $(LOCAL_REGISTRY)

##@ Patching Runtime to specified state

.PHONY: set-runtime-state
set-runtime-state:
	./scripts/set_runtime_state.sh $(RUNTIME_ID) $(STATE)

.PHONY: create-kubeconfig-secret
create-kubeconfig-secret:
	./scripts/create_kubeconfig_secret.sh $(RUNTIME_ID)

##@ Patching Kyma to specified state

.PHONY: set-kyma-state
set-kyma-state:
	./scripts/set_kyma_state.sh $(KYMA_ID) $(STATE)

##@ Creating GardenerCluster resource

.PHONY: create-gardener-cluster
create-gardener-cluster:
	./scripts/create_gardener_cluster_cr.sh $(GLOBAL_ACCOUNT_ID)

.PHONY: generate-env-docs
generate-env-docs:
	pip install -r scripts/python/requirements.txt
	python3 scripts/python/generate_env_docs.py
	python3 scripts/python/generate_values_doc.py
