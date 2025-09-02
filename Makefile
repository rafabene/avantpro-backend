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

.PHONY: all build test clean run deps tidy swagger help

# Default target
all: clean deps tidy swagger test build

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

# Run all tests
test:
	@echo "Running all tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./... || true
	@echo ""
	@echo "Coverage Summary:"
	$(GOCMD) tool cover -func=coverage.out | tail -1
	@echo ""
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "Open coverage.html in your browser to view the detailed report"

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

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	$(GOCMD) run $(MAIN_PATH)/main.go

# Run with hot reload (requires air)
dev: swagger
	@echo "Starting development server with hot reload..."
	@which air > /dev/null || (echo "air not installed. Run: make install-tools" && exit 1)
	air

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

# Full check (format, imports, vet, lint, test)
check: fmt fix-imports vet lint test
	@echo "All checks passed!"

# Release build (optimized)
release: clean deps tidy swagger test
	@echo "Building release version..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags "-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) -v $(MAIN_PATH)
	@echo "Release build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Test:"
	@echo "  all           - Clean, deps, tidy, swagger, test, and build"
	@echo "  build         - Build the application"
	@echo "  build-linux   - Build for Linux"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
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
	@echo "  check         - Run all code quality checks"
	@echo ""
	@echo "Development:"
	@echo "  run           - Run the application"
	@echo "  dev           - Generate Swagger docs and run with hot reload (air)"
	@echo "  swagger       - Generate Swagger documentation"
	@echo "  install-tools - Install all development tools"
	@echo ""
	@echo "Database:"
	@echo "  db/setup      - Start PostgreSQL container"
	@echo "  db/teardown   - Stop and remove PostgreSQL container"
	@echo "  db/status     - Check PostgreSQL container status"
	@echo "  db/logs       - Show PostgreSQL container logs"
	@echo "  db/shell      - Connect to PostgreSQL shell"
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