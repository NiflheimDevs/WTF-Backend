.PHONY: help api worker build test fmt tidy clean api-document river

APP_NAME ?= wtf-backend
BIN_DIR ?= bin
API_ENTRYPOINT := ./cmd/api
WORKER_ENTRYPOINT := ./cmd/worker
DATABASE_URL := postgresql://postgres:123456@localhost:5432/WTF
DOC_SUFFIX ?=

help:
	@echo "Available targets:"
	@echo "  make api                     - Run the API server"
	@echo "  make worker                  - Run the worker"
	@echo "  make build                   - Build the API binary"
	@echo "  make test                    - Run all tests"
	@echo "  make fmt                     - Format Go files"
	@echo "  make tidy                    - Tidy Go modules"
	@echo "  make api-document            - Build API docs"
	@echo "  make api-document DOC_SUFFIX=v1  - Build versioned docs"
	@echo "  make river                   - Install river and run migrations"
	@echo "  make clean                   - Remove build artifacts"

api:
	go run $(API_ENTRYPOINT)

worker:
	go run $(WORKER_ENTRYPOINT)

api-document:
	npx @redocly/cli build-docs ./API-Document/openapi.yaml -o ./API-Document/api-docs$(DOC_SUFFIX).html

river:
	go install github.com/riverqueue/river/cmd/river@latest
	river migrate up --database-url $(DATABASE_URL)

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(API_ENTRYPOINT)

test:
	go test -run=^$$ ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
