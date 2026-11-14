.PHONY: all build test test-coverage build-web clean run-mock studio

# Variables
BINARY_NAME=bin/ragfuzz
WEB_DIR=web

all: build-web build test

build-web:
	cd $(WEB_DIR) && npm install && npm run build

build:
	go build -o $(BINARY_NAME) ./cmd/ragfuzz

test:
	go test -v -race ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

run-mock:
	go run ./examples/mock_rag_server.go -port 8080 -corpus ./examples/corpus

studio: build
	./$(BINARY_NAME) studio --port 4040

clean:
	rm -rf bin/ coverage.out coverage.html internal/server/dist/ junit.xml report.html ragfuzz-report.json
