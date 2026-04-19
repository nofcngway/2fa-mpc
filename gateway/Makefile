.PHONY: generate build run

generate:
	./scripts/generate.sh

build:
	go build -o bin/gateway ./cmd/app

run: build
	./bin/gateway
