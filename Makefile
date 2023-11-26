# Install all development tools and build artifacts to the project's `bin` directory.
export GOBIN=$(CURDIR)/bin

install-hooks: ## Install git hooks
	@sh ./scripts/install-hooks.sh


.PHONY: install-tools
install-tools: $(GOBIN) ## Install all tools into bin directory.
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	@go install mvdan.cc/gofumpt@v0.5.0
	@go install github.com/bufbuild/buf/cmd/buf@v1.28.1
	@go install github.com/envoyproxy/protoc-gen-validate@v1.0.2

	@go install golang.org/x/tools/cmd/goimports
	@go install github.com/golang/mock/mockgen
	@go install google.golang.org/protobuf/cmd/protoc-gen-go
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
	@go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc

.PHONY: build
build: ## Builds all services in this repository.
	go install ./cmd/services/...

.PHONY: build-languages
build-languages: ## Builds the container languages or language
	@go run ./cmd/tools/container-builder/main.go

.PHONY: build-languages/verbose
build-languages/verbose: ## Builds the container languages or language with verbose mode enabled.
	@go run ./cmd/tools/container-builder/main.go -v

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(GOBIN)

.PHONY: generate
generate: install-tools ## Generate mocks, florence features and other code.
	@go generate ./...
	@$(MAKE) fmt

.PHONY: fmt
fmt: install-tools ## Format code.
	@$(GOBIN)/goimports -w -local "github.com/stephensli/" $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: lint
lint: install-tools ## Lint code.
	golangci-lint run --config ./build/.golangci.yml ./...


.PHONY: generate-proto
generate-proto: install-tools ## Generate protobufs.
	bash ./proto/gen.sh
	@$(MAKE) fmt

.PHONY: lint-proto
lint-proto: install-tools ## Lint protobufs.
	$(GOBIN)/buf lint

.PHONY: test
test: ## Run all tests.
	go test -race ./...

.PHONY: test-coverage
test-coverage: ## Run all tests and check test coverage
	@go test -coverprofile=coverage.out ./... ; \
	cat coverage.out | \
	awk 'BEGIN {cov=0; stat=0;} $$3!="" { cov+=($$3==1?$$2:0); stat+=$$2; } \
	END {printf("Total coverage: %.2f%% of statements\n", (cov/stat)*100);}'
	@go tool cover -html=coverage.out

.PHONY: tidy
tidy:  ## Tidies-up the code and modules.
	@go mod tidy

.PHONY: help
help:
	@grep -E '^[/a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
