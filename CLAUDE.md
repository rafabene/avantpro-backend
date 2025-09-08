# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Development Workflow
- `make all` - Complete development workflow: clean, deps, tidy, swagger, test, build
- `make dev` - Start development server with hot reload using Air
- `make run` - Run the application directly with go run
- `make install-tools` - Install all development tools (swag, golangci-lint, air, goimports)

### Code Quality
- `make check` - Run all code quality checks (fmt, fix-imports, vet, lint)
- `make lint` - Run golangci-lint
- `make fix-imports` - Organize imports with goimports using local prefix
- `make swagger` - Generate Swagger documentation
- `make test` - Run tests with Ginkgo framework and mock generation

### Database
- `make db/setup` - Start PostgreSQL container for development
- `make db/teardown` - Stop and remove PostgreSQL container
- `make db/shell` - Connect to PostgreSQL shell
- `make db/populate-test` - Clear tables and create test data with user rafabene@gmail.com/123456

## Architecture Overview

### Strict Layer Isolation Architecture
The codebase implements a sophisticated three-layer architecture with **complete layer isolation**:

**Controllers → Services → Repositories**

#### Layer Isolation Implementation
**Critical Pattern**: Each layer has its own type definitions and cannot import model types from other layers:

- **Repository Layer** (`internal/models/`): Contains domain models used only by repositories, includes GORM hooks and business logic methods
- **Service Layer** (`internal/services/`): 
  - `types.go`: Centralized shared type definitions across services
  - `*_models.go`: Service-specific DTOs per service with no imports from other layers
- **Controller Layer** (`internal/controllers/`):
  - `types.go`: Centralized shared type definitions across controllers 
  - `*_models.go`: Controller-specific DTOs with Swagger annotations and no imports from other layers

#### Type Conversion Between Layers
Each controller implements conversion functions to bridge layer boundaries:
```go
// Example conversion pattern in controllers
func (c *AuthController) toServiceLoginRequest(req *ServiceLoginRequest) *services.LoginRequest {
    return &services.LoginRequest{
        Email:    req.Email,
        Password: req.Password,
    }
}
```

### Key Design Patterns

**Complete Layer Isolation**: No cross-package imports for model types. Controllers cannot import `models` or `services` types; services cannot import `models` types.

**Type Conversion at Boundaries**: Conversion functions in each layer transform types when calling other layers, maintaining strict isolation.

**Interface-First Design**: All major components define interfaces that are implemented by concrete types. Repository and service interfaces are fully documented.

**Domain Models vs DTOs**: Clear separation between internal domain models (`models.User`) with GORM hooks and API DTOs (`controllers.UserResponse`) with Swagger annotations.

### Core Features

#### Authentication & Authorization
- **JWT Authentication**: Token-based auth with configurable expiration and refresh
- **Password Security**: bcrypt encryption with GORM hooks for automatic hashing
- **Account Lockout**: Configurable failed login attempts with IP-based lockout
- **Password Reset**: Token-based password reset flow

#### Organization Management System
Complete multi-tenant organization system:
- **Organization CRUD**: Create, read, update, delete with role-based permissions
- **Member Management**: Admin/user roles with role-based access control
- **Invitation System**: Email-based invitations with expiration and token acceptance
- **Permission Hierarchy**: Creator always admin, role-based feature access

#### Notification System
Organization-scoped notification preferences:
- **Real-time Notifications**: WebSocket-based notification delivery
- **Organization Preferences**: Notification settings per organization (not per user)
- **Event Types**: Member actions, invitation events, organization updates
- **Test Notifications**: Generate test notifications for preference validation

### Data Architecture

#### Database Design
- **PostgreSQL** with GORM ORM and UUID primary keys using `gen_random_uuid()`
- **Soft Deletes**: GORM DeletedAt for user and profile data
- **AutoMigrate**: Automatic schema management with UUID extension setup
- **Foreign Key Relationships**: Properly cascaded relationships between entities

#### Security Implementation
- **SQL Injection Prevention**: Repository layer validates sorting fields with whitelists
- **Trusted Proxies**: Environment-based proxy configuration for production
- **Input Validation**: Comprehensive validation using go-playground/validator
- **RFC 7807 Error Handling**: Standardized problem details for HTTP APIs

### Testing Strategy

#### Testcontainers Integration
- **Real PostgreSQL**: Tests run against actual PostgreSQL instances via testcontainers
- **Automatic Fallback**: Tests skip when Docker is unavailable
- **Mock Generation**: Automatic mock generation for repository interfaces using mockgen
- **Ginkgo Framework**: BDD-style testing with comprehensive integration tests

#### Test Organization
- **Integration Tests**: Full service layer tests with real database
- **Repository Tests**: Data layer tests with testcontainers
- **Mock-based Tests**: Service tests with mocked repositories
- **19 Test Specs**: Comprehensive coverage including authentication, security, and business logic

## Business Logic Patterns

### Authentication Flow
1. **Registration**: Create user account with optional profile, automatic password hashing
2. **Login**: JWT token generation with user data and security logging
3. **Account Lockout**: IP-based lockout after configurable failed attempts
4. **Password Reset**: Token-based reset with expiration validation

### Organization Multi-tenancy
1. **Creation**: User creates organization, becomes admin automatically
2. **Member Management**: Admins invite users via email tokens
3. **Invitation Flow**: Email token → user acceptance → member creation
4. **Role Management**: Creator cannot be removed, admins manage organization

### Notification Preferences
1. **Organization-Scoped**: Preferences stored per organization, not per user
2. **Event-Based**: Different notification types for various organization events
3. **Real-time Delivery**: WebSocket-based notification broadcasting
4. **Bulk Operations**: Mass preference updates with validation

## Configuration & Environment

### Environment Variables
Required configuration in `.env`:
- `ENV`: "development" or "production" (affects Swagger availability)
- `PORT`: Server port (default: 8080)
- `DB_*`: PostgreSQL connection parameters
- `JWT_SECRET`: **Critical to change in production**
- `JWT_EXPIRES_IN`: Token expiration duration
- `TRUSTED_PROXIES`: Production proxy IPs

### Security Configuration
- **Development Mode**: Swagger UI available, no trusted proxies
- **Production Mode**: Swagger disabled, configurable trusted proxies
- **CORS**: Configured for `http://localhost:4200` (adjust for production)
- **Database SSL**: Use `DB_SSLMODE=require` in production

## API Documentation & Usage

### Swagger Integration
- **Automatic Generation**: Documentation generated from code annotations
- **Development Access**: Available at `/swagger/index.html` in development mode
- **Type Safety**: All controller DTOs fully documented with examples

### Header Requirements
- **Authorization**: `Bearer <JWT_TOKEN>` for authenticated endpoints
- **Organization-ID**: Required for organization-scoped operations
- **User-ID**: Automatically injected by middleware from JWT

### Example Usage Patterns
```bash
# Organization-scoped operations require Organization-ID header
curl -H "Authorization: Bearer <token>" \
     -H "Organization-ID: <uuid>" \
     http://localhost:8080/api/v1/organizations/members

# Pagination and sorting support
curl "http://localhost:8080/api/v1/organizations/my?page=1&limit=10&sortBy=name&sortOrder=asc"
```

## Development Guidelines

### Code Quality Requirements
- **Mandatory Testing**: Always run `make test` and `make check` after modifications
- **Layer Isolation**: Never create cross-layer imports for model types
- **Import Organization**: Use `goimports` with local prefix `github.com/rafabene/avantpro-backend`
- **Type Conversions**: Implement conversion functions at layer boundaries

### Architecture Constraints
- **No Cross-Layer Imports**: Controllers cannot import `models` or `services` types
- **Service Isolation**: Services cannot import `models` types
- **Repository Domain**: Only repositories use domain models from `models` package
- **Interface Dependencies**: All layers depend on interfaces, not concrete implementations

### Key Business Rules
- **Username as Email**: Username field must be valid email address
- **Multi-tenant Design**: Application is organization-scoped for all business operations
- **Creator Privileges**: Organization creator always has admin role and cannot be removed
- **Notification Scope**: Preferences are organization-level, not user-level
- **Password Policy**: Enforced minimum requirements with bcrypt encryption

## Development Workflow Notes

### Hot Reload Development
- Server runs continuously with `make dev` using Air
- Automatic Swagger generation on startup
- No need to restart server for code changes

### Database Management
- PostgreSQL container management via Docker
- Test data population with known credentials
- Automatic schema migration via GORM AutoMigrate
- SQL scripts for data management in `sql/` directory