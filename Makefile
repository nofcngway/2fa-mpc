.DEFAULT_GOAL := help
.PHONY: help up down build-all test-all lint-all

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

up: ## Start full system via docker-compose
	docker compose up -d --build

down: ## Stop full system and remove containers
	docker compose down

build-all: ## Build all service Docker images
	docker build -t mpc-2fa-auth -f auth/Dockerfile auth/
	docker build -t mpc-2fa-mpc -f mpc/Dockerfile mpc/
	docker build -t mpc-2fa-twofa -f twofa/Dockerfile .

test-all: ## Run tests for all services
	cd auth && go test ./... -count=1
	cd twofa && go test ./... -count=1
	cd mpc && go test ./... -count=1

lint-all: ## Run linters for all services
	cd auth && go vet ./...
	cd twofa && go vet ./...
	cd mpc && go vet ./...
