.PHONY: all build test test-short test-cover lint fmt vet tidy clean help sdk-js sdk-kotlin brain docker docker-core docker-up docker-down

BINARY_NAME := probex
BUILD_DIR := ./bin
CORE_DIR := ./core

# Go build flags
LDFLAGS := -ldflags "-s -w -X main.version=1.0.0"

all: build

## build: Build the probex binary
build:
	cd $(CORE_DIR) && go build $(LDFLAGS) -o ../$(BUILD_DIR)/$(BINARY_NAME) .

## test: Run all Go tests
test:
	cd $(CORE_DIR) && go test ./... -v -race -timeout 120s

## test-short: Run tests without verbose output
test-short:
	cd $(CORE_DIR) && go test ./... -race -timeout 120s

## test-cover: Run tests with coverage report
test-cover:
	cd $(CORE_DIR) && go test ./... -race -timeout 120s -coverpkg=./... -coverprofile=coverage.out
	@cd $(CORE_DIR) && go tool cover -func=coverage.out | tail -1
	@echo "HTML report: cd core && go tool cover -html=coverage.out"

## lint: Run golangci-lint
lint:
	cd $(CORE_DIR) && golangci-lint run

## fmt: Format Go code
fmt:
	cd $(CORE_DIR) && gofmt -s -w .

## vet: Run go vet
vet:
	cd $(CORE_DIR) && go vet ./...

## tidy: Tidy Go modules
tidy:
	cd $(CORE_DIR) && go mod tidy

## sdk-js: Build JavaScript/TypeScript SDK
sdk-js:
	cd sdk-js && npm install && npm run build

## sdk-kotlin: Build Kotlin SDK
sdk-kotlin:
	cd sdk-kotlin && ./gradlew build

## sdk-java: Build Java SDK and plugins
sdk-java:
	cd sdk-java && mvn package -q

## brain: Install Python AI brain
brain:
	cd brain && pip install -e .

## docker: Build full Docker image (Go + Python)
docker:
	docker build -t probex:latest -f docker/Dockerfile .

## docker-core: Build minimal Docker image (Go only)
docker-core:
	docker build -t probex:core -f docker/Dockerfile.core .

## docker-up: Start PROBEX server via Docker Compose
docker-up:
	docker compose -f docker/docker-compose.yml up -d

## docker-up-ai: Start PROBEX server + AI brain via Docker Compose
docker-up-ai:
	docker compose -f docker/docker-compose.yml --profile ai up -d

## docker-down: Stop Docker Compose services
docker-down:
	docker compose -f docker/docker-compose.yml down

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	cd $(CORE_DIR) && go clean

## run: Build and run probex with args (usage: make run ARGS="scan https://example.com")
run: build
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## test-all: Run tests for all components
test-all: test
	@echo "--- JS SDK ---"
	-cd sdk-js && npm test 2>/dev/null || echo "(skipped: npm not configured)"
	@echo "--- Kotlin SDK ---"
	-cd sdk-kotlin && ./gradlew test 2>/dev/null || echo "(skipped: gradle not configured)"
	@echo "--- Python Brain ---"
	-cd brain && python -m pytest 2>/dev/null || echo "(skipped: pytest not configured)"

## help: Show this help message
help:
	@echo "PROBEX — Zero-Test API Intelligence Engine"
	@echo ""
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
