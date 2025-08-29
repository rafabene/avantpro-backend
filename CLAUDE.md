# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Development Workflow
- `make all` - Complete development workflow: clean, deps, tidy, swagger, test, build
- `make dev` - Start development server with hot reload using Air
- `make run` - Run the application directly with go run
- `make install-tools` - Install all development tools (swag, golangci-lint, air, goimports)

### Testing
- `make test` - Run all tests (includes testcontainers for PostgreSQL)
- `make test-coverage` - Run tests with coverage report, generates coverage.html
- `go test -v ./internal/repositories` - Run repository tests only (uses testcontainers)
- `go test -v ./tests/integration` - Run integration tests only (uses testcontainers)
- `go test -v ./internal/services` - Run service tests only (uses mocks)

### Code Quality
- `make check` - Run all code quality checks (fmt, fix-imports, vet, lint, test)
- `make lint` - Run golangci-lint
- `make fix-imports` - Organize imports with goimports using local prefix
- `make swagger` - Generate Swagger documentation

### Database
- `make db/setup` - Start PostgreSQL container for development
- `make db/teardown` - Stop and remove PostgreSQL container
- `make db/shell` - Connect to PostgreSQL shell
- Tests automatically use testcontainers and don't require manual database setup

## Architecture Overview

### Three-Layer Architecture
The codebase follows a clean three-layer architecture pattern:

**Controllers → Services → Repositories**

- **Controllers** (`internal/controllers/`): Handle HTTP requests, validate input, convert between DTOs and domain models
- **Services** (`internal/services/`): Business logic, validation, coordinate between repositories
- **Repositories** (`internal/repositories/`): Data access layer, database operations using GORM

### Key Design Patterns

**Dependency Injection**: Each layer depends on interfaces, not concrete implementations. Controllers depend on service interfaces, services depend on repository interfaces.

**Interface-First Design**: All major components define interfaces that are implemented by concrete types. Repository and service interfaces are fully documented.

**Domain Models vs DTOs**: Clear separation between internal domain models (`models.User`, `models.Profile`) and API request/response DTOs (`UserCreateRequest`, `UserResponse`).

**Password Security**: All user passwords are automatically encrypted using bcrypt with GORM hooks. The `Password` field is never returned in API responses.

**Sorting and Pagination**: All list endpoints support sorting with `sortBy` and `sortOrder` parameters. Repository layer validates allowed fields to prevent SQL injection.

### Error Handling
Uses RFC 7807 Problem Details for HTTP APIs via the `moogar0880/problems` library. Custom error handling in `internal/errors/problems.go` with specific error types for validation, not found, conflict, and internal errors.

### Testing Strategy

**Multi-Level Testing**:
- **Unit Tests**: Services use mocks for repository dependencies
- **Repository Tests**: Use PostgreSQL testcontainers for real database testing
- **Integration Tests**: Full HTTP API tests with testcontainers

**Testcontainers**: Repository and integration tests use PostgreSQL testcontainers. Tests automatically skip when Docker is unavailable with graceful degradation.

## API Features

### User Management
The API provides complete CRUD operations for users with the following features:

- **User Creation**: Creates users with encrypted passwords and optional profile
- **Find by Username**: Special endpoint to find users by username (email)
- **Profile Management**: Users can have detailed address and contact information
- **Soft Deletes**: Users are soft deleted using GORM's DeletedAt

### Data Models
- **User**: Contains username (email), name, encrypted password, and optional profile
- **Profile**: Contains complete address (street, city, district, zip_code) and phone number
- **Validation**: Comprehensive validation using go-playground/validator
- **Security**: Passwords are automatically hashed using bcrypt before saving

### Sorting Implementation
List endpoints support sorting with query parameters:
- `sortBy`: Field name (name, username, createdAt, updatedAt)
- `sortOrder`: Direction (asc, desc, defaults to desc)
- Automatic field validation and SQL injection prevention
- CamelCase to snake_case normalization (createdAt → created_at)
- Default sorting: created_at DESC

### Example Usage
```
# Create user with profile
POST /api/v1/users
{
  "username": "user@example.com",
  "name": "John Doe", 
  "password": "password123",
  "profile": {
    "street": "123 Main St",
    "city": "São Paulo",
    "district": "Centro", 
    "zip_code": "01234567",
    "phone": "11987654321"
  }
}

# Find by username
GET /api/v1/users/username/user@example.com

# List with sorting
GET /api/v1/users?sortBy=name&sortOrder=asc&page=1&limit=10
```

### Configuration
Environment-based configuration in `internal/config/` supports development and production modes with different database connection pools, logging levels, Gin modes, and trusted proxy configuration.

### Security Features
- **Password Encryption**: All passwords automatically encrypted with bcrypt using GORM hooks
- **Trusted Proxies**: Gin router configured with environment-based proxy trust
  - Development: No trusted proxies for security (`SetTrustedProxies(nil)`)
  - Production: Configurable via `TRUSTED_PROXIES` environment variable
- **SQL Injection Prevention**: Repository layer validates sorting fields with whitelist
- **RFC 7807 Error Handling**: Standardized error responses
- **Input Validation**: Comprehensive validation using struct tags and go-playground/validator

### Database
- **PostgreSQL** with GORM ORM
- **UUID Primary Keys** using PostgreSQL's `gen_random_uuid()`
- **Foreign Key Relationships** between User and Profile with cascade options
- **Soft Deletes** via GORM's DeletedAt for both User and Profile
- **AutoMigrate** used throughout for schema management
- **Testcontainers** for testing instead of SQLite

### API Documentation
Swagger/OpenAPI documentation generated automatically from code annotations using swaggo/swag. Access at `/swagger/index.html` in development mode.

## Project Structure Notes

### Internal Package Organization
- `cmd/server/` - Application entry point with dependency wiring
- `internal/config/` - Environment-based configuration management
- `internal/database/` - Database connection and migration utilities with UUID support
- `internal/errors/` - RFC 7807 error handling and validation formatting
- `internal/models/` - User and Profile domain models with encrypted password support
- `internal/repositories/` - User repository with GetByUsername method
- `internal/services/` - User service with comprehensive validation and business logic
- `internal/controllers/` - User HTTP controllers with full CRUD and username lookup
- `tests/integration/` - Full API integration tests with testcontainers

### Environment Variables
The application uses godotenv for development and supports these key variables:
- `ENV` - "development" or "production"
- `PORT` - Server port (default: 8080)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database configuration
- `TRUSTED_PROXIES` - Comma-separated list of trusted proxy IPs for production

### Import Organization
Use `goimports` with local prefix `github.com/rafabene/avantpro-backend` to organize imports correctly.

### Key Business Rules
- **Username is Email**: The username field must be a valid email address
- **Password Minimum Length**: Passwords must be at least 6 characters
- **Unique Usernames**: Email addresses must be unique across all users
- **Profile is Optional**: Users can be created without a profile initially
- **Address Validation**: When profile is provided, all address fields are required
- **Phone Normalization**: Phone numbers are normalized by removing formatting characters
- **Case Sensitivity**: Usernames are stored and searched in lowercase

### Testing Requirements
- All new features must include unit tests for services
- Repository operations must be tested with testcontainers
- API endpoints must have integration tests
- Password encryption must be verified in tests
- Username uniqueness must be tested
- Profile relationships must be tested