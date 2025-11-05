# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**AvantPro Backend** - Go REST API for subscription management following Clean Architecture principles.

- **Language**: Go 1.25+
- **Framework**: Gin (HTTP), GORM (ORM)
- **Database**: PostgreSQL 16+ with golang-migrate
- **Module**: `github.com/rafabene/avantpro-backend`

## Architecture

This codebase follows **Clean Architecture** with strict layer separation:

### Layer Structure (Dependency Flow: Inward)

```
Presentation → Service → Domain ← Infrastructure
```

**Domain Layer** (`internal/domain/`)
- Pure business entities, value objects, and interfaces
- **Zero external dependencies** - no frameworks, no database code
- Entities contain business rules and validation logic
- Repositories and gateways are **interfaces only**
- Example: `entities.User` has methods like `IsAdmin()`, `Validate()`, `SoftDelete()`

**Service Layer** (`internal/services/`)
- Orchestrates business logic and use cases
- Coordinates repositories and external gateways
- Manages transactions via `UnitOfWork` pattern
- Example: `UserService.CreateUser()` validates, checks duplicates, creates user, sends email

**Infrastructure Layer** (`internal/infrastructure/`)
- Implements domain interfaces (repositories, gateways)
- Database connections, external API clients, configs
- **Key separation**: GORM models (`UserModel`) vs domain entities (`User`)
- Mapper functions convert between models and entities

**Presentation Layer** (`internal/handlers/http/`)
- HTTP handlers using Gin framework
- DTOs for request/response serialization
- Thin controllers - delegate to services immediately

### Critical Pattern: Model ↔ Entity Separation

**GORM Models** live in `infrastructure/persistence/postgres/models.go`:
```go
type UserModel struct {
    ID           string  `gorm:"type:uuid;primary_key"`
    Email        string  `gorm:"type:varchar(255);uniqueIndex"`
    DeletedAt    *int64  // Unix timestamp for soft delete
}
```

**Domain Entities** live in `domain/entities/`:
```go
type User struct {
    ID        string
    Email     valueobjects.Email  // Value Object, not string
    DeletedAt *time.Time         // time.Time, not Unix
}
```

Repositories provide `toModel()` and `toEntity()` converters. **Never mix GORM models with domain logic**.

## Soft Delete Pattern

**All entities support soft delete** - see implementation in:
- Domain: `entities.User.SoftDelete()`, `IsDeleted()`, `Restore()`
- Repository: All queries filter `WHERE deleted_at IS NULL`
- Database: `deleted_at BIGINT` column (Unix timestamp)

When adding new entities:
1. Add `DeletedAt *time.Time` to entity
2. Add `deleted_at BIGINT` to migration
3. Add soft delete methods to entity
4. Filter deleted records in repository queries

## Development Commands

### Quick Start
```bash
make deps            # Download Go dependencies
make install-tools   # Install Air, golangci-lint, migrate, govulncheck
make docker/up       # Start PostgreSQL + Redis
make db/migrate-up   # Apply database migrations
make dev             # Start with hot reload (Air)
```

### Database Operations
```bash
make db/migration name=create_users_table  # Create new migration files
make db/migrate-up                         # Apply all pending migrations
make db/migrate-down                       # Rollback last migration
make db/migrate-version                    # Show current migration version
make db/reset                              # Drop, recreate, and migrate
```

**Migration files**: `internal/infrastructure/persistence/migrations/`
- Format: `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`
- Use SQL, not GORM auto-migrate
- Always create reversible migrations (`.down.sql`)

### Testing
```bash
make test/unit        # Fast unit tests with mocks
make test/integration # Integration tests with testcontainers
make test/e2e         # End-to-end HTTP tests
make test/coverage    # Generate coverage report (coverage.html)
```

**Test structure**:
- Unit tests alongside code: `user_service_test.go`
- Integration tests: `tests/integration/`
- Mocks: `tests/mocks/` (testify/mock)

### Code Quality
```bash
make pre-commit   # Run fmt + lint + unit tests (run before commits)
make lint         # golangci-lint v2.6.1
make lint-fix     # Auto-fix linting issues
make fmt          # Format code with gofmt
make vulncheck    # Check for vulnerabilities (govulncheck)
```

## Key Patterns & Conventions

### Error Handling

Domain errors are **message IDs for i18n**, not user-facing strings:
```go
// internal/domain/errors/errors.go
var ErrUserNotFound = errors.New("error.user_not_found")
```

Always use `errors.Is()` for comparison, never `==`:
```go
// ✅ Correct
if errors.Is(err, gorm.ErrRecordNotFound) { }

// ❌ Wrong
if err == gorm.ErrRecordNotFound { }
```

HTTP handlers translate errors to RFC 7807 Problem Details via i18n.

### Context Keys (Transaction Support)

Use **custom types** for context keys to avoid collisions:
```go
type contextKey string
const txKey contextKey = "tx"

ctx = context.WithValue(ctx, txKey, tx)  // ✅
ctx = context.WithValue(ctx, "tx", tx)   // ❌ staticcheck error
```

Repositories extract transactions from context:
```go
func (r *UserRepository) getDB(ctx context.Context) *gorm.DB {
    if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
        return tx  // Use transaction if present
    }
    return r.db  // Otherwise use regular DB
}
```

### Value Objects

Email, CPF, and other validated types are **Value Objects**, not primitives:
```go
// internal/domain/valueobjects/email.go
type Email struct { value string }

func NewEmail(email string) (Email, error) {
    // Normalize and validate
    return Email{value: normalized}, nil
}
```

Use in entities to enforce validation at construction time.

### Dependency Injection

Main function (`cmd/api/main.go`) wires dependencies manually:
```go
db := postgres.NewDatabaseConnection(cfg)
userRepo := postgres.NewUserRepository(db)
userService := services.NewUserService(userRepo, uow, logger)
userHandler := httphandlers.NewUserHandler(userService)
```

No DI framework - keep it explicit and simple.

### Configuration

Uses Viper for multi-source config:
```go
// internal/infrastructure/config/config.go
cfg, _ := config.Load()  // Reads .env, env vars, config files
```

Environment variables override config files. See `.env.example` for schema.

## Internationalization (i18n)

**Message IDs** in code, translations in JSON:
```go
// Code
i18nSvc.T(ctx, "error.user_not_found")

// en.json
{"error.user_not_found": "User not found"}

// pt-BR.json
{"error.user_not_found": "Usuário não encontrado"}
```

Middleware auto-detects language from:
1. Query param: `?lang=pt-BR`
2. Accept-Language header
3. Default: `en`

Locale files: `internal/infrastructure/i18n/locales/*.json`

## Adding New Features

Follow this flow:
1. **Domain**: Create entity, value objects, repository interface
2. **Infrastructure**: Implement repository, create migration
3. **Service**: Add use case business logic
4. **Handler**: Add HTTP endpoint, DTOs
5. **Routes**: Register in `cmd/api/main.go`
6. **Tests**: Unit (with mocks) → Integration (testcontainers) → E2E

Example sequence for "Products" feature:
```bash
# 1. Migration
make db/migration name=create_products_table
# Edit .up.sql and .down.sql

# 2. Domain
touch internal/domain/entities/product.go
touch internal/domain/repositories/product_repository.go

# 3. Infrastructure
touch internal/infrastructure/persistence/postgres/product_repository.go
touch internal/infrastructure/persistence/postgres/models.go  # Add ProductModel

# 4. Service
touch internal/services/product_service.go

# 5. Handler
touch internal/handlers/http/product_handler.go
touch internal/handlers/dto/product_dto.go

# 6. Wire in main.go
# Add routes, DI
```

## Security & Best Practices

- **HTTP Server**: Always set `ReadHeaderTimeout` to prevent Slowloris attacks
- **Input Validation**: Use `binding` tags in DTOs, validate in domain entities
- **SQL Injection**: GORM parameterizes queries, but review raw SQL in migrations
- **Secrets**: Never commit `.env`, use environment variables in production
- **Authentication**: JWT stored in Authorization header (not yet implemented)

## Common Gotchas

1. **Import Aliases**: `http` stdlib conflicts with local package - use `httphandlers` alias
2. **Soft Delete**: All new repositories must filter `deleted_at IS NULL`
3. **GORM vs Entity**: Never put GORM tags on domain entities
4. **Time Types**: Models use `int64` (Unix), entities use `time.Time`
5. **Error Wrapping**: Use `fmt.Errorf("context: %w", err)` to preserve error chains

## Documentation

Specs organized by type in `specs/` directory:

**Functional Requirements** (what the system does):
- `functional/auth.md` - Use cases, business rules, RBAC flows

**Technical Specs** (how it works):
- `technical/architecture.md` - Clean Architecture deep dive
- `technical/security.md` - JWT/OAuth2 implementation (not yet coded)
- `technical/database.md` - Migrations, GORM patterns
- `technical/validation-i18n.md` - Validation and i18n
- `technical/testing.md` - Testing strategies

**Development Guides**:
- `development/tooling.md` - Air, Docker, Makefile, golangci-lint

## Current Status

**Implemented**:
- Clean Architecture foundation
- User entity with RBAC (admin/user/guest roles)
- Soft delete pattern
- i18n with pt-BR, en, es
- PostgreSQL with migrations
- GORM repository pattern
- Unit tests with testify
- Hot reload with Air
- Linting with golangci-lint v2

**Not Yet Implemented**:
- Authentication (JWT/OAuth2)
- Email/SMS gateways
- Payment integration
- Subscription entities
- Redis caching
- Background jobs
- Swagger documentation
- Integration/E2E tests
