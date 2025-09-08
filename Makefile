# Variables
APP_NAME=avantpro-backend
MAIN_PATH=./cmd/server
BUILD_DIR=./bin
DOCKER_IMAGE=avantpro-backend
VERSION?=latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=$(APP_NAME)
BINARY_UNIX=$(BINARY_NAME)_unix

.PHONY: all build clean run deps tidy swagger test test-coverage generate db/populate-test db/clear db/status-tables db/run-sql help

# Default target
all: clean deps tidy generate swagger test build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for Linux
build-linux:
	@echo "Building $(APP_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) -v $(MAIN_PATH)
	@echo "Linux build complete: $(BUILD_DIR)/$(BINARY_UNIX)"


# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOGET) -v ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	swag init -g $(MAIN_PATH)/main.go

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Fix and organize imports
fix-imports:
	@echo "Fixing and organizing imports..."
	@which goimports > /dev/null || (echo "goimports not installed. Run: make install-tools" && exit 1)
	goimports -w -local github.com/rafabene/avantpro-backend .


# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Run tests
test: generate
	@echo "Running tests..."
	@which ginkgo > /dev/null || (echo "ginkgo CLI not installed. Run: make install-tools" && exit 1)
	ginkgo -v -r ./internal/

# Run integration tests with testcontainers
test-integration:
	@echo "Running integration tests with testcontainers..."
	@which ginkgo > /dev/null || (echo "ginkgo CLI not installed. Run: make install-tools" && exit 1)
	TESTCONTAINERS_RYUK_DISABLED=true ginkgo -v -r ./internal/

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"


# Run the application
run:
	@echo "Running $(APP_NAME)..."
	$(GOCMD) run $(MAIN_PATH)/main.go

# Run with hot reload (requires air)
dev: swagger
	@echo "Starting development server with hot reload..."
	@which air > /dev/null || (echo "air not installed. Run: make install-tools" && exit 1)
	air

# Generate mocks
generate:
	@echo "Generating mocks..."
	@which mockgen > /dev/null || (echo "mockgen not installed. Run: make install-tools" && exit 1)
	@mkdir -p internal/services/tests/mocks
	$(GOCMD) generate ./internal/repositories/...
	@echo "Mocks generated successfully!"

# Install development tools
install-tools:
	@echo "Installing essential development tools..."
	@echo "Installing Swagger generator (required for API docs)..."
	$(GOCMD) install github.com/swaggo/swag/cmd/swag@latest
	@echo "Installing golangci-lint (code quality)..."
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing air (hot reload for development)..."
	$(GOCMD) install github.com/air-verse/air@latest
	@echo "Installing goimports (import organizer)..."
	$(GOCMD) install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing Ginkgo CLI (testing framework)..."
	$(GOCMD) install github.com/onsi/ginkgo/v2/ginkgo@v2.25.3
	@echo "Installing mockgen (mock generator)..."
	$(GOCMD) install go.uber.org/mock/mockgen@latest
	@echo "Development tools installed successfully!"

# Database setup with Docker
db/setup:
	@echo "Starting PostgreSQL container..."
	docker run -d \
		--name avantpro-backend-postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=avantpro_backend \
		-p 5432:5432 \
		postgres:15-alpine
	@echo "PostgreSQL container started!"
	@echo "Connection string: postgresql://postgres:postgres@localhost:5432/avantpro_backend"

# Database teardown
db/teardown:
	@echo "Stopping and removing PostgreSQL container..."
	docker stop avantpro-backend-postgres || true
	docker rm avantpro-backend-postgres || true
	@echo "PostgreSQL container removed!"

# Database status
db/status:
	@echo "Checking PostgreSQL container status..."
	@docker ps -f name=avantpro-backend-postgres --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Database logs
db/logs:
	@echo "PostgreSQL container logs:"
	docker logs avantpro-backend-postgres

# Database shell
db/shell:
	@echo "Connecting to PostgreSQL..."
	docker exec -it avantpro-backend-postgres psql -U postgres -d avantpro_backend

# Clear all tables using SQL script
db/clear:
	@echo "Clearing all database tables..."
	@docker exec -i avantpro-backend-postgres psql -U postgres -d avantpro_backend < sql/truncate_all_tables.sql
	@echo "All tables cleared!"

# Show database table status
db/status-tables:
	@echo "Database table status:"
	@docker exec -i avantpro-backend-postgres psql -U postgres -d avantpro_backend < sql/show_table_status.sql

# Run SQL script manually
db/run-sql:
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make db/run-sql FILE=path/to/script.sql"; \
		echo "Example: make db/run-sql FILE=sql/truncate_all_tables.sql"; \
		exit 1; \
	fi
	@echo "Running SQL script: $(FILE)"
	@docker exec -i avantpro-backend-postgres psql -U postgres -d avantpro_backend < $(FILE)

# Populate database with test data
db/populate-test:
	@echo "Populating database with test data..."
	@echo "Clearing existing tables using SQL script..."
	@docker exec -i avantpro-backend-postgres psql -U postgres -d avantpro_backend < sql/truncate_all_tables.sql > /dev/null 2>&1
	@echo "Tables cleared successfully!"
	@echo "Creating test data..."
	@docker exec -i avantpro-backend-postgres psql -U postgres -d avantpro_backend < sql/populate_test_data.sql > /dev/null 2>&1 && echo "✅ Test data created successfully!" || echo "❌ Failed to create test data"
	@echo "Verifying test data..."
	@docker exec -i avantpro-backend-postgres psql -U postgres -d avantpro_backend < sql/verify_test_data.sql
	@echo ""
	@echo "🎉 Test data populated successfully!"
	@echo "📧 Test user: rafabene@gmail.com"
	@echo "🔑 Password: 123456"
	@echo "🏢 Organizations: AvantPro Tecnologia, Consultoria Rafael"

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(VERSION)


# Check for outdated dependencies
check-updates:
	@echo "Checking for outdated dependencies..."
	$(GOCMD) list -u -m all

# Full check (format, imports, vet, lint)
check: fmt fix-imports vet lint
	@echo "All checks passed!"

# Release build (optimized)
release: clean deps tidy swagger
	@echo "Building release version..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags "-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) -v $(MAIN_PATH)
	@echo "Release build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  all           - Clean, deps, tidy, swagger, and build"
	@echo "  build         - Build the application"
	@echo "  build-linux   - Build for Linux"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps          - Install dependencies"
	@echo "  tidy          - Tidy dependencies"
	@echo "  check-updates - Check for outdated dependencies"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format code"
	@echo "  fix-imports   - Fix and organize imports (goimports)"
	@echo "  vet           - Vet code"
	@echo "  lint          - Lint code (golangci-lint)"
	@echo "  generate      - Generate mocks (auto-called by test)"
	@echo "  test          - Run tests (auto-generates mocks)"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  check         - Run all code quality checks"
	@echo ""
	@echo "Development:"
	@echo "  run           - Run the application"
	@echo "  dev           - Generate Swagger docs and run with hot reload (air)"
	@echo "  swagger       - Generate Swagger documentation"
	@echo "  install-tools - Install all development tools"
	@echo ""
	@echo "Database:"
	@echo "  db/setup        - Start PostgreSQL container"
	@echo "  db/teardown     - Stop and remove PostgreSQL container"
	@echo "  db/status       - Check PostgreSQL container status"
	@echo "  db/logs         - Show PostgreSQL container logs"
	@echo "  db/shell        - Connect to PostgreSQL shell"
	@echo "  db/clear        - Clear all tables using SQL script"
	@echo "  db/status-tables - Show table record counts and status"
	@echo "  db/run-sql      - Run SQL script (usage: make db/run-sql FILE=script.sql)"
	@echo "  db/populate-test - Clear tables and create test user (rafabene@gmail.com/123456)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo ""
	@echo "Release:"
	@echo "  release       - Build optimized release version"
	@echo ""
	@echo "Help:"
	@echo "  help          - Show this help message"