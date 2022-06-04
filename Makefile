install-tools: ## Install all tools into bin directory.
	@cat build/tools.go | grep "_" | awk '{print $$2}' | xargs go install

.PHONY: build
build: ## Builds all services in this repository.
	go install ./cmd/services/...

.PHONY: build-docker-images
build-docker-images: ## Builds all the required docker images
	./build/dockerfiles/update-docker.sh

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(GOBIN)

.PHONY: lint
lint: install-tools ## Lint code.
	golangci-lint run --config ./build/.golangci.yml ./...

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
