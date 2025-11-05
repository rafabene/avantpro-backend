# Ferramentas e Desenvolvimento

**Versão**: 2.0
**Data**: 05/11/2025

---

## 1. Visão Geral

Ferramentas essenciais para produtividade:
- **Air**: Hot reload durante desenvolvimento
- **Docker**: Containerização e ambiente consistente
- **Makefile**: Automação de tarefas
- **golangci-lint**: Linter agregador para qualidade
- **Swagger**: Documentação interativa de API
- **VSCode/GoLand**: IDEs recomendadas

---

## 2. Air (Hot Reload)

### 2.1 Instalação

```bash
# Via go install
go install github.com/cosmtrek/air@latest

# Verificar instalação
air -v
```

### 2.2 Configuração (.air.toml)

```toml
# .air.toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
# Comando de build
cmd = "go build -o ./tmp/main ./cmd/api"

# Binário gerado
bin = "tmp/main"

# Delay antes de re-build (ms)
delay = 1000

# Excluir diretórios
exclude_dir = ["assets", "tmp", "vendor", "testdata", "tests", "docs"]

# Excluir arquivos
exclude_file = []

# Extensões para watch
include_ext = ["go", "tpl", "tmpl", "html"]

# Diretórios para não watch
exclude_regex = ["_test\\.go"]

# Parar em erro de build
stop_on_error = true

# Log de build
send_interrupt = false
kill_delay = 500

[log]
time = true

[color]
main = "magenta"
watcher = "cyan"
build = "yellow"
runner = "green"

[misc]
# Limpar tela ao rebuild
clean_on_exit = true
```

### 2.3 Uso

```bash
# Iniciar development server com hot reload
air

# Com arquivo de config custom
air -c .air.custom.toml
```

---

## 3. Docker e Docker Compose

### 3.1 Dockerfile

```dockerfile
# Dockerfile

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Instalar dependências do sistema
RUN apk add --no-cache git

# Copiar go.mod e go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fonte
COPY . .

# Build da aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copiar binário do build stage
COPY --from=builder /app/main .

# Copiar configs e migrations
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/internal/infrastructure/persistence/migrations ./migrations
COPY --from=builder /app/internal/infrastructure/i18n/locales ./locales

# Expor porta
EXPOSE 8080

# Rodar aplicação
CMD ["./main"]
```

### 3.2 Docker Compose (Desenvolvimento)

```yaml
# docker-compose.yml
version: '3.8'

services:
  # PostgreSQL
  postgres:
    image: postgres:16-alpine
    container_name: avantpro-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: avantpro
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis
  redis:
    image: redis:7-alpine
    container_name: avantpro-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # API (development)
  api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: avantpro-api
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASS=postgres
      - DB_NAME=avantpro
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=your-secret-key
      - ENV=development
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      # Hot reload em desenvolvimento
      - .:/app
    command: air

volumes:
  postgres_data:
  redis_data:
```

### 3.3 Docker Compose (Produção)

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: avantpro-postgres-prod
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASS}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - postgres_prod_data:/var/lib/postgresql/data
    restart: always

  redis:
    image: redis:7-alpine
    container_name: avantpro-redis-prod
    volumes:
      - redis_prod_data:/data
    restart: always

  api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: avantpro-api-prod
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASS=${DB_PASS}
      - DB_NAME=${DB_NAME}
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - ENV=production
    depends_on:
      - postgres
      - redis
    restart: always

volumes:
  postgres_prod_data:
  redis_prod_data:
```

### 3.4 Comandos Docker

```bash
# Build
docker-compose build

# Iniciar serviços
docker-compose up -d

# Ver logs
docker-compose logs -f api

# Parar serviços
docker-compose down

# Parar e remover volumes
docker-compose down -v

# Executar comando dentro do container
docker-compose exec api sh

# Rodar migrations
docker-compose exec api ./migrate-up.sh
```

---

## 4. Makefile Completo

```makefile
# Makefile

# Configs
APP_NAME := avantpro-backend
VERSION := $(shell git describe --tags --always --dirty)
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
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2}' $(MAKEFILE_LIST)

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
.PHONY: test
test: ## Run all tests
	$(GOTEST) ./... -v

.PHONY: test-unit
test-unit: ## Run unit tests
	$(GOTEST) ./internal/... -v -short

.PHONY: test-integration
test-integration: ## Run integration tests
	$(GOTEST) ./tests/integration/... -v

.PHONY: test-e2e
test-e2e: ## Run e2e tests
	$(GOTEST) ./tests/e2e/... -v

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	$(GOTEST) ./... -coverprofile=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Database - Migrations
.PHONY: db/migration
db/migration: ## Create new migration (name=xxx)
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make db/migration name=create_users_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

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

.PHONY: lint-fix
lint-fix: ## Run linter and fix issues
	golangci-lint run --fix ./...

## Formatting
.PHONY: fmt
fmt: ## Format code
	$(GOCMD) fmt ./...
	gofumpt -l -w .

.PHONY: fmt-check
fmt-check: ## Check code formatting
	@test -z $(shell gofmt -l .) || (echo "Code needs formatting" && exit 1)

## Docker
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(APP_NAME):$(VERSION) .

.PHONY: docker-up
docker-up: ## Start Docker Compose services
	docker-compose up -d

.PHONY: docker-down
docker-down: ## Stop Docker Compose services
	docker-compose down

.PHONY: docker-logs
docker-logs: ## Show Docker Compose logs
	docker-compose logs -f api

## Swagger
.PHONY: swagger
swagger: ## Generate Swagger docs
	swag init -g cmd/api/main.go -o docs

.PHONY: swagger-fmt
swagger-fmt: ## Format Swagger comments
	swag fmt

## Database
.PHONY: db-create
db-create: ## Create database
	createdb -h $(DB_HOST) -U $(DB_USER) $(DB_NAME)

.PHONY: db-drop
db-drop: ## Drop database
	dropdb -h $(DB_HOST) -U $(DB_USER) $(DB_NAME)

.PHONY: db-reset
db-reset: db-drop db-create migrate-up ## Reset database

.PHONY: db-seed
db-seed: ## Seed database with test data
	$(GOCMD) run scripts/seed.go

## CI/CD
.PHONY: ci
ci: deps lint test ## Run CI pipeline

.PHONY: pre-commit
pre-commit: fmt lint test-unit ## Run pre-commit checks

## Install tools
.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install mvdan.cc/gofumpt@latest
	@echo "Tools installed successfully"

.DEFAULT_GOAL := help
```

---

## 5. golangci-lint

### 5.1 Configuração (.golangci.yml)

```yaml
# .golangci.yml

run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - errcheck          # Check error returns
    - gosimple          # Simplify code
    - govet             # Go vet
    - ineffassign       # Detect ineffectual assignments
    - staticcheck       # Static analysis
    - unused            # Unused code
    - gofmt             # Gofmt checks
    - goimports         # Manage imports
    - misspell          # Spell checking
    - unconvert         # Unnecessary type conversions
    - goconst           # Repeated strings to constants
    - gocyclo           # Cyclomatic complexity
    - dupl              # Code cloning
    - gocritic          # Go critic
    - gosec             # Security issues
    - bodyclose         # HTTP response body close
    - noctx             # HTTP request without context
    - sqlclosecheck     # SQL rows/stmt close
    - rowserrcheck      # SQL rows.Err check
    - errorlint         # Error wrapping
    - exportloopref     # Loop variable reference
    - exhaustive        # Switch exhaustiveness
    - nilerr            # Return nil error

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true

  govet:
    check-shadowing: true

  gocyclo:
    min-complexity: 15

  dupl:
    threshold: 100

  goconst:
    min-len: 3
    min-occurrences: 3

  misspell:
    locale: US

  gosec:
    excludes:
      - G104  # Errors unhandled (já cobre errcheck)

  gocritic:
    enabled-tags:
      - diagnostic
      - experimental
      - opinionated
      - performance
      - style

issues:
  exclude-rules:
    # Exclude test files from some linters
    - path: _test\.go
      linters:
        - gocyclo
        - errcheck
        - dupl
        - gosec

    # Exclude wire generated files
    - path: wire_gen\.go
      linters:
        - unused
        - deadcode

  max-issues-per-linter: 0
  max-same-issues: 0

output:
  format: colored-line-number
  print-issued-lines: true
  print-linter-name: true
```

### 5.2 Uso

```bash
# Rodar linter
golangci-lint run ./...

# Com fix automático
golangci-lint run --fix ./...

# Apenas arquivos modificados
golangci-lint run --new-from-rev=HEAD~1

# Output em JSON
golangci-lint run --out-format=json ./...
```

---

## 6. Swagger (swaggo/swag)

### 6.1 Setup

```go
// cmd/api/main.go
package main

import (
    "github.com/gin-gonic/gin"
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"

    _ "avantpro-backend/docs"  // Swagger docs gerados
)

// @title           AvantPro API
// @version         1.0
// @description     API para gerenciamento de assinaturas

// @contact.name    API Support
// @contact.email   support@avantpro.com

// @host            localhost:8080
// @BasePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
    router := gin.Default()

    // Swagger UI
    router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // Routes
    setupRoutes(router)

    router.Run(":8080")
}
```

### 6.2 Annotations

```go
// internal/handlers/http/user_handler.go

// CreateUser godoc
// @Summary      Create a new user
// @Description  Creates a new user account with email and password
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreateUserRequest  true  "User creation data"
// @Success      201      {object}  dto.UserResponse
// @Failure      400      {object}  dto.ErrorResponse       "Validation error"
// @Failure      409      {object}  dto.ErrorResponse       "Email already exists"
// @Failure      500      {object}  dto.ErrorResponse       "Internal error"
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
    // Implementation
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Returns a single user by ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID (UUID)"
// @Success      200  {object}  dto.UserResponse
// @Failure      404  {object}  dto.ErrorResponse  "User not found"
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
    // Implementation
}
```

### 6.3 Gerar Docs

```bash
# Gerar documentação Swagger
make swagger

# Ou diretamente
swag init -g cmd/api/main.go -o docs

# Acessar UI
# http://localhost:8080/swagger/index.html
```

---

## 7. Ambiente e Configuração

### 7.1 Arquivo .env

```bash
# .env

# Environment
ENV=development

# Server
PORT=8080
HOST=0.0.0.0

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=postgres
DB_NAME=avantpro
DB_MAX_CONNS=25
DB_MIN_CONNS=5

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-super-secret-key-change-in-production
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# OAuth2
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
OAUTH_REDIRECT_URL=http://localhost:8080

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password

# Logging
LOG_LEVEL=debug

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

### 7.2 .env.example

```bash
# .env.example
# Copy this to .env and fill with your values

ENV=development
PORT=8080

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=
DB_NAME=avantpro

REDIS_URL=redis://localhost:6379

JWT_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
```

### 7.3 .gitignore

```gitignore
# Binaries
/bin/
/tmp/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out
coverage.html

# Dependency directories
/vendor/

# Go workspace file
go.work

# Environment variables
.env
.env.local
.env.*.local

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Air tmp
/tmp/
```

---

## 8. Scripts Úteis

### 8.1 Seed Database

```go
// scripts/seed.go
package main

import (
    "context"
    "log"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/infrastructure/persistence/postgres"
    // ...
)

func main() {
    // Setup database connection
    db := setupDatabase()

    // Create repositories
    userRepo := postgres.NewUserRepository(db)

    ctx := context.Background()

    // Seed users
    users := []entities.User{
        {Email: "admin@example.com", Name: "Admin", Role: entities.RoleAdmin},
        {Email: "user@example.com", Name: "User", Role: entities.RoleUser},
    }

    for _, user := range users {
        if err := userRepo.Create(ctx, &user); err != nil {
            log.Printf("Failed to create user %s: %v", user.Email, err)
        } else {
            log.Printf("Created user: %s", user.Email)
        }
    }

    log.Println("Seeding completed!")
}
```

### 8.2 Health Check Script

```bash
#!/bin/bash
# scripts/health-check.sh

API_URL=${1:-http://localhost:8080}

response=$(curl -s -o /dev/null -w "%{http_code}" $API_URL/health)

if [ $response -eq 200 ]; then
    echo "✓ API is healthy"
    exit 0
else
    echo "✗ API is not healthy (HTTP $response)"
    exit 1
fi
```

---

## 9. CI/CD com GitHub Actions

### 9.1 Test Pipeline

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  test:
    name: Run Tests
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Download dependencies
        run: go mod download

      - name: Run linter
        run: |
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
          golangci-lint run ./...

      - name: Run tests
        env:
          DB_URL: postgresql://test:test@localhost:5432/testdb?sslmode=disable
          REDIS_URL: redis://localhost:6379
        run: |
          go test ./... -v -coverprofile=coverage.out
          go tool cover -func=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

### 9.2 Build and Deploy

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    name: Build and Deploy
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Build Docker image
        run: docker build -t avantpro-backend:${{ github.sha }} .

      - name: Push to registry
        run: |
          echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker tag avantpro-backend:${{ github.sha }} your-registry/avantpro-backend:latest
          docker push your-registry/avantpro-backend:latest

      # Deploy to your platform (AWS, GCP, etc)
```

---

## 10. VSCode Configuration

### 10.1 settings.json

```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.formatTool": "gofumpt",
  "editor.formatOnSave": true,
  "go.testFlags": ["-v"],
  "go.coverOnSave": true,
  "go.coverageDecorator": {
    "type": "gutter"
  }
}
```

### 10.2 extensions.json

```json
{
  "recommendations": [
    "golang.go",
    "ms-azuretools.vscode-docker",
    "humao.rest-client",
    "eamodio.gitlens"
  ]
}
```

---

**Versão**: 2.0
**Última Atualização**: 05/11/2025
