.DEFAULT_GOAL := help
.PHONY: help generate mock build run test clean generate-keys tidy docker-build docker-run lint

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

generate: ## Generate protobuf code
	bash scripts/generate.sh

mock: ## Generate mock files (minimock)
	minimock -i github.com/vbncursed/vkr/auth/internal/services/auth_service.Storage -o internal/services/auth_service/mocks/ -s "_mock.go"
	minimock -i github.com/vbncursed/vkr/auth/internal/services/auth_service.SessionStorage -o internal/services/auth_service/mocks/ -s "_mock.go"
	minimock -i github.com/vbncursed/vkr/auth/internal/services/auth_service.EventProducer -o internal/services/auth_service/mocks/ -s "_mock.go"

build: generate ## Build the service binary
	go build -o bin/auth ./cmd/app

run: build ## Build and run the service
	./bin/auth

test: ## Run all tests
	go test ./... -count=1 -v

clean: ## Remove build artifacts
	rm -rf bin/ internal/pb/

generate-keys: ## Generate RSA key pair for JWT
	mkdir -p keys
	openssl genrsa -out keys/private.pem 2048
	openssl rsa -in keys/private.pem -pubout -out keys/public.pem
	@echo "RSA keys generated in keys/"

tidy: ## Tidy Go modules
	go mod tidy

docker-build: ## Build Docker image
	docker build -t mpc-2fa-auth .

docker-run: docker-build ## Run service in Docker (requires external infra)
	docker run --rm mpc-2fa-auth

lint: ## Run linter (go vet + golangci-lint if available)
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || true
