.DEFAULT_GOAL := help
.PHONY: help generate mock build run test clean tidy docker-build docker-run lint

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

generate: ## Generate protobuf code
	bash scripts/generate.sh

mock: ## Generate mock files (minimock)
	go generate ./internal/services/twofaService/...

build: generate ## Build the service binary
	go build -o bin/twofa ./cmd/app

run: build ## Build and run the service
	./bin/twofa

test: ## Run all tests
	go test ./... -count=1 -v

clean: ## Remove build artifacts
	rm -rf bin/ internal/pb/

tidy: ## Tidy Go modules
	go mod tidy

docker-build: ## Build Docker image (uses project root as context)
	cd .. && docker build -t mpc-2fa-twofa -f twofa/Dockerfile .

docker-run: docker-build ## Run service in Docker (requires external infra)
	docker run --rm mpc-2fa-twofa

lint: ## Run linter (go vet + golangci-lint if available)
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || true
