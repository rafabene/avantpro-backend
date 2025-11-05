# Makefile

# Configs
APP_NAME := avantpro-backend
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go configs
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Database
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASS ?= postgres
DB_NAME ?= avantpro
DB_URL := postgresql://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

MIGRATIONS_DIR := internal/infrastructure/persistence/migrations

# Colors
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: help
help: ## Show this help
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_\/\-]+:.*?## / {printf "  ${YELLOW}%-25s${GREEN}%s${RESET}\n", $$1, $$2}' $(MAKEFILE_LIST)

## Development
.PHONY: dev
dev: ## Run with hot reload (Air)
	air

.PHONY: run
run: ## Run application
	$(GOBUILD) -o $(GOBIN)/$(APP_NAME) ./cmd/api
	$(GOBIN)/$(APP_NAME)

.PHONY: build
build: ## Build application
	$(GOBUILD) -ldflags="-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -o $(GOBIN)/$(APP_NAME) ./cmd/api

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf $(GOBIN)
	rm -rf tmp/
	rm -rf coverage.*

## Dependencies
.PHONY: deps
deps: ## Download dependencies
	$(GOMOD) download

.PHONY: deps-update
deps-update: ## Update dependencies
	$(GOGET) -u ./...
	$(GOMOD) tidy

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	$(GOMOD) verify

## Tests
.PHONY: test/all
test/all: ## Run all tests
	$(GOTEST) ./... -v

.PHONY: test/unit
test/unit: ## Run unit tests
	$(GOTEST) ./internal/... -v -short

.PHONY: test/integration
test/integration: ## Run integration tests
	$(GOTEST) ./tests/integration/... -v

.PHONY: test/e2e
test/e2e: ## Run e2e tests
	$(GOTEST) ./tests/e2e/... -v

.PHONY: test/coverage
test/coverage: ## Run tests with coverage
	$(GOTEST) ./... -coverprofile=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Database
.PHONY: db/create
db/create: ## Create database
	createdb -h $(DB_HOST) -U $(DB_USER) $(DB_NAME)

.PHONY: db/drop
db/drop: ## Drop database
	dropdb -h $(DB_HOST) -U $(DB_USER) $(DB_NAME)

.PHONY: db/reset
db/reset: db/drop db/create db/migrate-up ## Reset database

.PHONY: db/migration
db/migration: ## Create new migration (name=xxx)
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make db/migration name=create_users_table"; \
		exit 1; \
	fi
	@mkdir -p $(MIGRATIONS_DIR)
	@NEXT=$$(printf "%06d" $$(ls $(MIGRATIONS_DIR) 2>/dev/null | grep -o '^[0-9]\+' | sort -n | tail -1 | awk '{print $$1 + 1}')); \
	if [ -z "$$NEXT" ] || [ "$$NEXT" = "000000" ]; then NEXT="000001"; fi; \
	UP_FILE="$(MIGRATIONS_DIR)/$${NEXT}_$(name).up.sql"; \
	DOWN_FILE="$(MIGRATIONS_DIR)/$${NEXT}_$(name).down.sql"; \
	echo "-- Migration: $(name)" > $$UP_FILE; \
	echo "" >> $$UP_FILE; \
	echo "-- Add your UP migration here" >> $$UP_FILE; \
	echo "" >> $$UP_FILE; \
	echo "-- Migration: $(name)" > $$DOWN_FILE; \
	echo "" >> $$DOWN_FILE; \
	echo "-- Add your DOWN migration here" >> $$DOWN_FILE; \
	echo "" >> $$DOWN_FILE; \
	echo "Created migration files:"; \
	echo "  - $$UP_FILE"; \
	echo "  - $$DOWN_FILE"

.PHONY: db/migrate-up
db/migrate-up: ## Apply migrations
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

.PHONY: db/migrate-down
db/migrate-down: ## Rollback last migration
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

.PHONY: db/migrate-version
db/migrate-version: ## Show current migration version
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

.PHONY: db/migrate-force
db/migrate-force: ## Force migration version (version=xxx)
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required. Usage: make db/migrate-force version=5"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(version)

## Linting
.PHONY: lint
lint: ## Run linter
	golangci-lint run ./...

.PHONY: vulncheck
vulncheck: ## Check for vulnerabilities
	govulncheck ./...

.PHONY: lint-fix
lint-fix: ## Run linter and fix issues
	golangci-lint run --fix ./...

## Formatting
.PHONY: fmt
fmt: ## Format code
	$(GOCMD) fmt ./...

.PHONY: fmt-check
fmt-check: ## Check code formatting
	@test -z $$(gofmt -l .) || (echo "Code needs formatting" && exit 1)

## Docker
.PHONY: docker/build
docker/build: ## Build Docker image
	docker build -t $(APP_NAME):$(VERSION) .

.PHONY: docker/up
docker/up: ## Start Docker Compose services
	docker-compose up -d

.PHONY: docker/down
docker/down: ## Stop Docker Compose services
	docker-compose down

.PHONY: docker/logs
docker/logs: ## Show Docker Compose logs
	docker-compose logs -f api

## CI/CD
.PHONY: pre-commit
pre-commit: fmt lint test/unit ## Run pre-commit checks

## Install tools
.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing tools..."
	$(GOCMD) install github.com/air-verse/air@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$($(GOCMD) env GOPATH)/bin
	$(GOCMD) install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	$(GOCMD) install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Tools installed successfully"

.DEFAULT_GOAL := help
