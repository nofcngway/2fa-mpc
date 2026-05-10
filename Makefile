.DEFAULT_GOAL := help
.PHONY: help up down build-all test-all lint-all generate-api

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

up: ## Start full system via docker-compose
	docker compose up -d

up-build: ## Start full system via docker-compose with build
	docker compose up -d --build

down: ## Stop full system and remove containers
	docker compose down

down-v: ## Stop full system and remove containers and volumes
	docker compose down -v

build-all: ## Build all service Docker images
	docker build -t mpc-2fa-auth -f auth/Dockerfile auth/
	docker build -t mpc-2fa-mpc -f mpc/Dockerfile mpc/
	docker build -t mpc-2fa-twofa -f twofa/Dockerfile .

test-all: ## Run tests for all services
	cd auth && go test ./... -count=1
	cd twofa && go test ./... -count=1
	cd mpc && go test ./... -count=1

lint-all: ## Run linters for all services (go vet)
	cd auth && go vet ./...
	cd twofa && go vet ./...
	cd mpc && go vet ./...
	cd gateway && go vet ./...

golangci-lint: ## Run golangci-lint v2 across all services (uses .golangci.yml)
	@which golangci-lint > /dev/null 2>&1 || (echo "install golangci-lint v2: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" && exit 1)
	cd auth && golangci-lint run --config ../.golangci.yml ./...
	cd twofa && golangci-lint run --config ../.golangci.yml ./...
	cd mpc && golangci-lint run --config ../.golangci.yml ./...
	cd gateway && golangci-lint run --config ../.golangci.yml ./...

generate-api: ## Generate protobuf code for all services
	@bash auth/scripts/generate.sh
	@bash twofa/scripts/generate.sh
	@bash mpc/scripts/generate.sh
	@bash gateway/scripts/generate.sh

# ── Load testing ─────────────────────────────────────────────
LOADTEST = docker compose -f docker-compose.yml -f loadtest/docker-compose.loadtest.yaml --profile loadtest

load-login: ## Run k6 login throughput scenario
	$(LOADTEST) run --rm k6 run /scripts/login.js

load-setup: ## Run k6 2FA setup scenario
	$(LOADTEST) run --rm k6 run /scripts/setup-2fa.js

load-verify: ## Run k6 2FA verify scenario
	$(LOADTEST) run --rm k6 run /scripts/verify-2fa.js

load-mixed: ## Run k6 mixed workload scenario (70/20/10)
	$(LOADTEST) run --rm k6 run /scripts/mixed.js

load-all: load-login load-setup load-verify load-mixed ## Run all k6 scenarios sequentially
