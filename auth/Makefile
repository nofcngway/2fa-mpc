.PHONY: generate generate-mocks build run test clean generate-keys tidy

generate:
	bash scripts/generate.sh

generate-mocks:
	minimock -i github.com/vbncursed/vkr/auth/internal/services/authService.Storage -o internal/services/authService/mocks/ -s "_mock.go"

build: generate
	go build -o bin/auth ./cmd/app

run: build
	./bin/auth

test:
	go test ./... -count=1 -v

clean:
	rm -rf bin/ internal/pb/

generate-keys:
	mkdir -p keys
	openssl genrsa -out keys/private.pem 2048
	openssl rsa -in keys/private.pem -pubout -out keys/public.pem
	@echo "RSA keys generated in keys/"

tidy:
	go mod tidy
