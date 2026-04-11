.PHONY: generate build run test clean tidy

generate:
	bash scripts/generate.sh

build: generate
	go build -o bin/twofa ./cmd/app

run: build
	./bin/twofa

test:
	go test ./... -count=1 -v

clean:
	rm -rf bin/ internal/pb/

tidy:
	go mod tidy
