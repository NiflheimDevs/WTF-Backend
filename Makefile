.PHONY: help run build test fmt tidy clean

APP_NAME ?= wtf-backend
BIN_DIR ?= bin
API_ENTRYPOINT := ./cmd/api
WORKER_ENTRYPOINT := ./cmd/worker
DATABASE_URL := "postgresql://postgres:123456@localhost:5432/WTF"

help:
	@echo "Available targets:"
	@echo "  make api    - Run the API server"
	@echo "  make build  - Build the API binary"
	@echo "  make test   - Run all tests"
	@echo "  make fmt    - Format Go files"
	@echo "  make tidy   - Tidy Go modules"
	@echo "  make clean  - Remove build artifacts"

api:
	go run $(API_ENTRYPOINT)

worker:
	go run $(WORKER_ENTRYPOINT)

river:
	go install github.com/riverqueue/river/cmd/river@latest

	river migrate up \
	--database-url $(DATABASE_URL)

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
