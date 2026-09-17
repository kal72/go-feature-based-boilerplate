## ─── Configuration ────────────────────────────────────────────────────────────
MODULE      := go-feature-based-boilerplate
BUILD_DIR   := bin
BINARY      := $(BUILD_DIR)/server
DATABASE_URL ?= postgres://appuser:apppassword@localhost:5432/appdb?sslmode=disable

## Proto settings
PROTO_DIR     := api/proto
GEN_DIR       := gen/pb
SWAGGER_DIR   := gen/openapi
PROTO_FILES   := $(shell find $(PROTO_DIR) -name '*.proto')

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## ─── Build ─────────────────────────────────────────────────────────────────────
.PHONY: build
build: ## Build the service binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/api
	@echo "built $(BINARY)"

.PHONY: run
run: ## Run the service locally
	go run ./cmd/api

## ─── Proto generation ──────────────────────────────────────────────────────────
.PHONY: protogen
protogen: ## Generate Go code + Swagger/OpenAPI from proto files (delegates to protogen script)
ifeq ($(OS),Windows_NT)
	scripts\protogen.bat
else
	./scripts/protogen.sh
endif

## ─── Wire ──────────────────────────────────────────────────────────────────────
.PHONY: wire
wire: ## Run Google Wire to regenerate wire_gen.go
	@which wire > /dev/null || go install github.com/google/wire/cmd/wire@latest
	wire ./bootstrap/...

## ─── Mocks ─────────────────────────────────────────────────────────────────────
.PHONY: mocks
mocks: ## Generate mocks via go generate (requires mockgen)
	@which mockgen > /dev/null || go install go.uber.org/mock/mockgen@latest
	go generate ./internal/...

## ─── Test ──────────────────────────────────────────────────────────────────────
.PHONY: test
test: ## Run unit tests
	go test -race -count=1 -timeout=60s ./bootstrap/... ./infrastructure/... ./internal/... ./pkg/...

.PHONY: test-verbose
test-verbose: ## Run unit tests with verbose output
	go test -race -count=1 -timeout=60s -v ./bootstrap/... ./infrastructure/... ./internal/... ./pkg/...

.PHONY: test-cover
test-cover: ## Run unit tests with coverage report
	go test -race -count=1 -coverprofile=coverage.out -covermode=atomic ./bootstrap/... ./infrastructure/... ./internal/... ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "coverage report: coverage.html"

.PHONY: integration-test
integration-test: ## Run integration tests (requires running infrastructure)
	go test -race -count=1 -timeout=120s -tags=integration ./tests/integration/...

.PHONY: e2e-test
e2e-test: ## Run end-to-end tests
	go test -race -count=1 -timeout=300s -tags=e2e ./tests/e2e/...

## ─── Lint ──────────────────────────────────────────────────────────────────────
.PHONY: lint
lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format all Go source files
	gofmt -w -s .
	goimports -w . 2>/dev/null || true

.PHONY: vet
vet: ## Run go vet
	go vet ./...

## ─── Database migrations ────────────────────────────────────────────────────────
.PHONY: migrate-up
migrate-up: ## Apply all pending migrations
	migrate -path migrations -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

.PHONY: migrate-status
migrate-status: ## Show current migration status
	migrate -path migrations -database "$(DATABASE_URL)" version

.PHONY: migration
migration: ## Create a new migration file (usage: make migration name=add_column_foo)
	@[ -n "$(name)" ] || (echo "usage: make migration name=<description>" && exit 1)
	@ts=$$(date +%Y%m%d%H%M%S); \
	touch migrations/$${ts}_$(name).up.sql migrations/$${ts}_$(name).down.sql; \
	echo "created migrations/$${ts}_$(name).up.sql and .down.sql"

## ─── Docker ────────────────────────────────────────────────────────────────────
.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build -f deployments/Dockerfile -t $(MODULE):latest .

.PHONY: docker-up
docker-up: ## Start all services via docker-compose
	docker compose -f deployments/docker-compose.yml up -d

.PHONY: docker-down
docker-down: ## Stop all services via docker-compose
	docker compose -f deployments/docker-compose.yml down

.PHONY: docker-logs
docker-logs: ## Tail application logs
	docker compose -f deployments/docker-compose.yml logs -f app

## ─── Dependency management ─────────────────────────────────────────────────────
.PHONY: deps
deps: ## Download and tidy Go modules
	go mod download
	go mod tidy

.PHONY: deps-upgrade
deps-upgrade: ## Upgrade all direct dependencies to latest minor/patch
	go get -u ./...
	go mod tidy

## ─── Scaffolding ────────────────────────────────────────────────────────────
.PHONY: new-feature
new-feature: ## Scaffold a new feature module (usage: make new-feature name=product)
	@./scripts/new-feature.sh "$(name)"

## ─── Clean ─────────────────────────────────────────────────────────────────────
.PHONY: clean
clean: ## Remove build artefacts
	rm -rf $(BUILD_DIR) coverage.out coverage.html

