# Especificação de Arquitetura - Backend Go

**Projeto**: AvantPro Backend
**Versão**: 2.0
**Data**: 05/11/2025
**Arquitetura**: Clean Architecture com Services

---

## 1. Visão Geral

Este documento define a arquitetura, padrões e convenções para aplicações backend em Go seguindo Clean Architecture com adaptações práticas.

### 1.1 Princípios Fundamentais

- **Separação de Responsabilidades**: Cada camada tem uma responsabilidade clara
- **Independência de Frameworks**: Domínio não depende de bibliotecas externas
- **Testabilidade**: Código facilmente testável com mocks e doubles
- **Dependency Inversion**: Dependências apontam para dentro (domínio)
- **Pragmatismo**: Arquitetura serve o projeto, não o contrário

### 1.2 Stack Tecnológico

| Categoria | Tecnologia | Justificativa |
|-----------|-----------|---------------|
| **Framework Web** | Gin | Performance, popularidade, middlewares abundantes |
| **Banco de Dados** | PostgreSQL | Robusto, ACID, extensível, JSON support |
| **ORM** | GORM | Produtividade, migrations, associations |
| **Migrações** | golang-migrate | SQL puro, controle total, produção-ready |
| **Autenticação** | JWT + OAuth2/OIDC | Stateless, escalável, login social |
| **Autorização** | RBAC customizado | Controle granular de permissões |
| **Validação** | go-playground/validator | Declarativa via tags, integra com Gin |
| **Logging** | slog (stdlib) | Nativo, estruturado, zero dependencies |
| **Configuração** | Viper | Flexível, env vars + arquivos |
| **i18n** | go-i18n | Arquivos JSON/YAML, pluralização |
| **Documentação** | Swagger/OpenAPI (swaggo) | UI interativa, gerada do código |
| **Testes** | testify + testcontainers | Assertions + containers reais |
| **Dev Tools** | Air, Docker, Makefile, golangci-lint | Produtividade e qualidade |

---

## 2. Arquitetura em Camadas

```
┌─────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                        │
│                  (HTTP Handlers/Controllers)                 │
│                                                              │
│  • Recebe requisições HTTP                                  │
│  • Validação de input (básica)                             │
│  • Chama Services                                           │
│  • Serializa responses                                      │
│  • Trata erros HTTP                                         │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     SERVICE LAYER                            │
│                  (Business Logic/Use Cases)                  │
│                                                              │
│  • Orquestra operações de negócio                          │
│  • Valida regras de domínio                                │
│  • Coordena repositories e gateways                        │
│  • Gerencia transações (UnitOfWork)                        │
│  • Implementa casos de uso                                  │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      DOMAIN LAYER                            │
│              (Entities, Value Objects, Interfaces)           │
│                                                              │
│  • Entidades de negócio (User, Order, Product)             │
│  • Value Objects (Email, Money, CPF)                        │
│  • Interfaces (Repositories, Gateways, Ports)              │
│  • Regras de negócio puras                                 │
│  • SEM dependências externas                                │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   INFRASTRUCTURE LAYER                       │
│              (Adapters, Implementations, External)           │
│                                                              │
│  • Implementações de Repositories (PostgreSQL)             │
│  • Implementações de Gateways (Email, SMS, Payment)        │
│  • Configuração de frameworks                               │
│  • Clients externos                                         │
│  • Database, cache, message queues                          │
└─────────────────────────────────────────────────────────────┘
```

### 2.1 Fluxo de Dados

```
HTTP Request
    ↓
[Middleware: Auth, Logging, CORS]
    ↓
[Handler] → valida input básico
    ↓
[Service] → orquestra lógica de negócio
    ↓
[Domain] → aplica regras de domínio
    ↓
[Repository] → persiste dados
    ↓
[Database] → PostgreSQL
    ↓
HTTP Response
```

---

## 3. Estrutura de Diretórios

```
avantpro-backend/
├── cmd/
│   └── api/
│       └── main.go                    # Entry point da aplicação
│
├── internal/
│   ├── domain/                        # DOMAIN LAYER (核心)
│   │   ├── entities/                  # Entidades de domínio
│   │   │   ├── user.go
│   │   │   ├── subscription.go
│   │   │   └── payment.go
│   │   │
│   │   ├── valueobjects/              # Value Objects
│   │   │   ├── email.go
│   │   │   ├── money.go
│   │   │   └── cpf.go
│   │   │
│   │   ├── repositories/              # Interfaces de persistência
│   │   │   ├── user_repository.go
│   │   │   ├── subscription_repository.go
│   │   │   └── payment_repository.go
│   │   │
│   │   ├── gateways/                  # Interfaces de serviços externos
│   │   │   ├── email_gateway.go
│   │   │   ├── payment_gateway.go
│   │   │   └── sms_gateway.go
│   │   │
│   │   ├── logger.go                  # Interface de logging
│   │   ├── unit_of_work.go            # Interface de transações
│   │   │
│   │   └── errors/                    # Erros de domínio
│   │       ├── errors.go
│   │       └── codes.go
│   │
│   ├── services/                      # SERVICE LAYER (casos de uso)
│   │   ├── user_service.go
│   │   ├── auth_service.go
│   │   ├── subscription_service.go
│   │   └── payment_service.go
│   │
│   ├── handlers/                      # PRESENTATION LAYER (HTTP)
│   │   ├── http/
│   │   │   ├── user_handler.go
│   │   │   ├── auth_handler.go
│   │   │   ├── subscription_handler.go
│   │   │   └── payment_handler.go
│   │   │
│   │   ├── middleware/                # Middlewares HTTP
│   │   │   ├── auth.go
│   │   │   ├── rbac.go
│   │   │   ├── logging.go
│   │   │   ├── cors.go
│   │   │   ├── rate_limit.go
│   │   │   └── i18n.go
│   │   │
│   │   └── dto/                       # DTOs (Request/Response)
│   │       ├── user_dto.go
│   │       ├── auth_dto.go
│   │       └── common.go
│   │
│   ├── infrastructure/                # INFRASTRUCTURE LAYER
│   │   ├── persistence/
│   │   │   ├── postgres/              # Implementações PostgreSQL
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── subscription_repository.go
│   │   │   │   ├── payment_repository.go
│   │   │   │   └── unit_of_work.go
│   │   │   │
│   │   │   └── migrations/            # SQL migrations
│   │   │       ├── 000001_create_users.up.sql
│   │   │       ├── 000001_create_users.down.sql
│   │   │       └── ...
│   │   │
│   │   ├── gateways/                  # Implementações de gateways
│   │   │   ├── email/
│   │   │   │   └── smtp_gateway.go
│   │   │   ├── payment/
│   │   │   │   └── stripe_gateway.go
│   │   │   └── sms/
│   │   │       └── twilio_gateway.go
│   │   │
│   │   ├── auth/                      # Autenticação/Autorização
│   │   │   ├── jwt.go
│   │   │   ├── oauth2.go
│   │   │   └── rbac.go
│   │   │
│   │   ├── logging/                   # Implementação de logging
│   │   │   └── slog_logger.go
│   │   │
│   │   ├── config/                    # Configuração
│   │   │   ├── config.go
│   │   │   └── viper.go
│   │   │
│   │   └── i18n/                      # Internacionalização
│   │       ├── i18n.go
│   │       └── locales/
│   │           ├── en.json
│   │           ├── pt-BR.json
│   │           └── es.json
│   │
│   └── pkg/                           # Pacotes compartilhados
│       ├── validator/                 # Validação customizada
│       ├── pagination/                # Helpers de paginação
│       ├── response/                  # Formatação de respostas
│       └── utils/                     # Utilities genéricas
│
├── tests/                             # Testes
│   ├── integration/                   # Integration tests
│   ├── e2e/                          # E2E tests
│   └── fixtures/                     # Test fixtures
│
├── docs/                              # Documentação
│   ├── api/                          # Swagger/OpenAPI specs
│   └── architecture/                 # Diagramas e specs
│
├── scripts/                           # Scripts úteis
│   ├── migration.sh
│   └── seed.sh
│
├── configs/                           # Arquivos de configuração
│   ├── config.yaml
│   ├── config.dev.yaml
│   └── config.prod.yaml
│
├── deployments/                       # Deploy configs
│   ├── docker/
│   │   └── Dockerfile
│   └── k8s/
│
├── .air.toml                         # Air config (hot reload)
├── .golangci.yml                     # Linter config
├── docker-compose.yml                # Docker services
├── Makefile                          # Task automation
├── go.mod
├── go.sum
└── README.md
```

---

## 4. Camadas Detalhadas

### 4.1 Domain Layer (`internal/domain/`)

**Responsabilidades:**
- Definir entidades de negócio
- Definir value objects
- Definir interfaces (contratos)
- Implementar regras de negócio puras

**Regras:**
- ❌ NÃO pode depender de camadas externas
- ❌ NÃO pode importar frameworks (Gin, GORM, etc)
- ❌ NÃO pode ter lógica de persistência ou HTTP
- ✅ PODE ter validações de domínio
- ✅ PODE ter métodos de negócio nas entidades

#### 4.1.1 Entities

```go
// internal/domain/entities/user.go
package entities

import (
    "time"
    "avantpro-backend/internal/domain/valueobjects"
)

type User struct {
    ID        string
    Email     valueobjects.Email
    Name      string
    Role      Role
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Métodos de domínio
func (u *User) IsAdmin() bool {
    return u.Role == RoleAdmin
}

func (u *User) CanAccessResource(resourceID string) bool {
    // Lógica de negócio pura
    return true
}
```

#### 4.1.2 Value Objects

```go
// internal/domain/valueobjects/email.go
package valueobjects

import (
    "errors"
    "regexp"
    "strings"
)

type Email struct {
    value string
}

func NewEmail(email string) (Email, error) {
    email = strings.TrimSpace(strings.ToLower(email))

    if !isValidEmail(email) {
        return Email{}, errors.New("invalid email format")
    }

    return Email{value: email}, nil
}

func (e Email) String() string {
    return e.value
}

func isValidEmail(email string) bool {
    pattern := `^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`
    matched, _ := regexp.MatchString(pattern, email)
    return matched
}
```

#### 4.1.3 Repositories (Interfaces)

```go
// internal/domain/repositories/user_repository.go
package repositories

import (
    "context"
    "avantpro-backend/internal/domain/entities"
)

type UserRepository interface {
    Create(ctx context.Context, user *entities.User) error
    FindByID(ctx context.Context, id string) (*entities.User, error)
    FindByEmail(ctx context.Context, email string) (*entities.User, error)
    Update(ctx context.Context, user *entities.User) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filters UserFilters) ([]*entities.User, error)
}

type UserFilters struct {
    Role     *entities.Role
    Page     int  // Página (começa em 1)
    PageSize int  // Itens por página (default: 20, max: 100)
}
```

#### 4.1.4 Gateways (Interfaces)

```go
// internal/domain/gateways/email_gateway.go
package gateways

import "context"

type EmailGateway interface {
    SendWelcomeEmail(ctx context.Context, email, name string) error
    SendPasswordResetEmail(ctx context.Context, email, token string) error
    SendVerificationEmail(ctx context.Context, email, code string) error
}
```

#### 4.1.5 Interfaces de Infraestrutura

**UnitOfWork** - Gerenciamento de transações:

```go
// internal/domain/unit_of_work.go
package domain

import "context"

type UnitOfWork interface {
    Begin(ctx context.Context) (context.Context, error)
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
    WithTransaction(ctx context.Context, fn func(context.Context) error) error
}
```

**Logger** - Interface de logging:

```go
// internal/domain/logger.go
package domain

type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
    Warn(msg string, args ...any)
    With(args ...any) Logger
}
```

### 4.2 Service Layer (`internal/services/`)

**Responsabilidades:**
- Implementar casos de uso (use cases)
- Orquestrar operações de negócio
- Coordenar repositories, gateways e interfaces de infraestrutura
- Gerenciar transações
- Aplicar regras de negócio complexas

**Regras:**
- ✅ PODE depender de `domain/`
- ✅ PODE usar múltiplos repositories/gateways
- ✅ DEVE usar UnitOfWork para transações
- ❌ NÃO pode conhecer detalhes HTTP (request/response)
- ❌ NÃO pode conhecer detalhes de banco (SQL)

```go
// internal/services/user_service.go
package services

import (
    "context"
    "avantpro-backend/internal/domain"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/repositories"
    "avantpro-backend/internal/domain/gateways"
)

type UserService struct {
    userRepo     repositories.UserRepository
    emailGateway gateways.EmailGateway
    uow          domain.UnitOfWork
    logger       domain.Logger
}

func NewUserService(
    userRepo repositories.UserRepository,
    emailGateway gateways.EmailGateway,
    uow domain.UnitOfWork,
    logger domain.Logger,
) *UserService {
    return &UserService{
        userRepo:     userRepo,
        emailGateway: emailGateway,
        uow:          uow,
        logger:       logger,
    }
}

// CreateUser cria um usuário e envia email de boas-vindas
func (s *UserService) CreateUser(ctx context.Context, input CreateUserInput) (*entities.User, error) {
    s.logger.Info("creating user", "email", input.Email)

    // Validar se email já existe
    existing, err := s.userRepo.FindByEmail(ctx, input.Email)
    if err == nil && existing != nil {
        return nil, ErrEmailAlreadyExists
    }

    // Criar entidade
    user := &entities.User{
        Email: input.Email,
        Name:  input.Name,
        Role:  entities.RoleUser,
    }

    // Executar em transação
    err = s.uow.WithTransaction(ctx, func(txCtx context.Context) error {
        // Salvar usuário
        if err := s.userRepo.Create(txCtx, user); err != nil {
            return err
        }

        // Enviar email (pode falhar sem rollback se desejado)
        if err := s.emailGateway.SendWelcomeEmail(txCtx, user.Email.String(), user.Name); err != nil {
            s.logger.Warn("failed to send welcome email", "error", err)
            // Não retorna erro - email é melhor esforço
        }

        return nil
    })

    if err != nil {
        s.logger.Error("failed to create user", "error", err)
        return nil, err
    }

    s.logger.Info("user created successfully", "user_id", user.ID)
    return user, nil
}

type CreateUserInput struct {
    Email string
    Name  string
}
```

### 4.3 Presentation Layer (`internal/handlers/`)

**Responsabilidades:**
- Receber requisições HTTP
- Validar input (formato, tipos)
- Chamar services apropriados
- Formatar responses
- Tratar erros e status codes HTTP

**Regras:**
- ✅ PODE depender de `services/` e `domain/entities`
- ✅ DEVE validar input com go-playground/validator
- ✅ DEVE retornar status codes HTTP apropriados
- ❌ NÃO pode ter lógica de negócio
- ❌ NÃO pode acessar repositories diretamente

```go
// internal/handlers/http/user_handler.go
package http

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/services"
    "avantpro-backend/internal/handlers/dto"
)

type UserHandler struct {
    userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
    return &UserHandler{
        userService: userService,
    }
}

// CreateUser godoc
// @Summary Create a new user
// @Description Creates a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "User data"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
    var req dto.CreateUserRequest

    // Bind e validação automática
    if err := c.ShouldBindJSON(&req); err != nil {
        // Formatar erros de validação com i18n
        validationErrors := formatValidationErrors(err)

        // RFC 7807 - Problem Details com i18n
        response := dto.ValidationErrorResponse(c, validationErrors)
        c.JSON(http.StatusBadRequest, response)
        return
    }

    // Chamar service
    user, err := h.userService.CreateUser(c.Request.Context(), services.CreateUserInput{
        Email: req.Email,
        Name:  req.Name,
    })

    if err != nil {
        // Tratar erros específicos com RFC 7807 + i18n
        var response dto.ErrorResponse

        switch err {
        case services.ErrEmailAlreadyExists:
            response = dto.ConflictErrorResponse(c, "error.conflict.email_exists")
            c.JSON(http.StatusConflict, response)
        default:
            response = dto.InternalErrorResponse(c)
            c.JSON(http.StatusInternalServerError, response)
        }
        return
    }

    // Retornar sucesso
    c.JSON(http.StatusCreated, dto.ToUserResponse(user))
}
```

#### 4.3.1 DTOs (Data Transfer Objects)

```go
// internal/handlers/dto/user_dto.go
package dto

import (
    "time"
    "avantpro-backend/internal/domain/entities"
)

type CreateUserRequest struct {
    Email string `json:"email" binding:"required,email"`
    Name  string `json:"name" binding:"required,min=2,max=100"`
}

type UserResponse struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    Role      string    `json:"role"`
    CreatedAt time.Time `json:"created_at"`
}

func ToUserResponse(user *entities.User) UserResponse {
    return UserResponse{
        ID:        user.ID,
        Email:     user.Email.String(),
        Name:      user.Name,
        Role:      string(user.Role),
        CreatedAt: user.CreatedAt,
    }
}

// ErrorResponse segue RFC 7807 (Problem Details for HTTP APIs)
// https://datatracker.ietf.org/doc/html/rfc7807
type ErrorResponse struct {
    Type     string                 `json:"type"`               // URI identificando o tipo de problema
    Title    string                 `json:"title"`              // Resumo curto e legível
    Status   int                    `json:"status"`             // HTTP status code
    Detail   string                 `json:"detail,omitempty"`   // Explicação específica deste erro
    Instance string                 `json:"instance,omitempty"` // URI da requisição que causou o erro
    Errors   []ValidationError      `json:"errors,omitempty"`   // Erros de validação (se aplicável)
    Meta     map[string]interface{} `json:"meta,omitempty"`     // Metadados adicionais
}
```

### 4.4 Infrastructure Layer (`internal/infrastructure/`)

**Responsabilidades:**
- Implementar interfaces definidas no domain
- Integrar com frameworks e bibliotecas
- Gerenciar conexões externas
- Configurar dependências

#### 4.4.1 Repository Implementation

```go
// internal/infrastructure/persistence/postgres/user_repository.go
package postgres

import (
    "context"
    "gorm.io/gorm"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/repositories"
)

type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) repositories.UserRepository {
    return &UserRepository{db: db}
}

// UserModel é o model GORM (separado da entidade de domínio)
type UserModel struct {
    ID        string `gorm:"primaryKey"`
    Email     string `gorm:"uniqueIndex;not null"`
    Name      string `gorm:"not null"`
    Role      string `gorm:"not null"`
    CreatedAt int64  `gorm:"autoCreateTime"`
    UpdatedAt int64  `gorm:"autoUpdateTime"`
}

func (UserModel) TableName() string {
    return "users"
}

func (r *UserRepository) Create(ctx context.Context, user *entities.User) error {
    model := r.toModel(user)

    db := r.getDB(ctx)
    if err := db.Create(model).Error; err != nil {
        return err
    }

    // Atualizar ID gerado
    user.ID = model.ID
    return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
    var model UserModel

    db := r.getDB(ctx)
    if err := db.Where("email = ?", email).First(&model).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }

    return r.toEntity(&model)
}

// getDB extrai DB do contexto (para suportar transações)
func (r *UserRepository) getDB(ctx context.Context) *gorm.DB {
    if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
        return tx
    }
    return r.db
}

// Conversores
func (r *UserRepository) toModel(user *entities.User) *UserModel {
    return &UserModel{
        ID:    user.ID,
        Email: user.Email.String(),
        Name:  user.Name,
        Role:  string(user.Role),
    }
}

func (r *UserRepository) toEntity(model *UserModel) (*entities.User, error) {
    email, err := valueobjects.NewEmail(model.Email)
    if err != nil {
        return nil, err
    }

    return &entities.User{
        ID:    model.ID,
        Email: email,
        Name:  model.Name,
        Role:  entities.Role(model.Role),
    }, nil
}
```

#### 4.4.2 Unit of Work Implementation

```go
// internal/infrastructure/persistence/postgres/unit_of_work.go
package postgres

import (
    "context"
    "gorm.io/gorm"
    "avantpro-backend/internal/domain"
)

type UnitOfWork struct {
    db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) domain.UnitOfWork {
    return &UnitOfWork{db: db}
}

func (uow *UnitOfWork) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
    tx := uow.db.Begin()

    // Adicionar transação ao contexto
    txCtx := context.WithValue(ctx, "tx", tx)

    // Executar função
    err := fn(txCtx)

    if err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit().Error
}
```

---

## 5. Patterns e Convenções

### 5.1 Nomenclatura

| Tipo | Padrão | Exemplo |
|------|--------|---------|
| **Entities** | PascalCase, singular | `User`, `Subscription`, `Payment` |
| **Repositories** | `<Entity>Repository` | `UserRepository`, `OrderRepository` |
| **Services** | `<Entity>Service` | `UserService`, `AuthService` |
| **Handlers** | `<Entity>Handler` | `UserHandler`, `ProductHandler` |
| **DTOs** | `<Action><Entity>Request/Response` | `CreateUserRequest`, `UserResponse` |
| **Gateways** | `<Service>Gateway` | `EmailGateway`, `PaymentGateway` |
| **Interfaces** | Nome descritivo (sem prefixo I) | `Logger`, `Cache`, `UnitOfWork` |
| **Métodos** | Verbos descritivos | `Create`, `Find`, `Update`, `Delete`, `List` |

### 5.2 Organização de Arquivos

- **Um tipo principal por arquivo**: `user.go` contém apenas `User`
- **Testes ao lado**: `user_test.go` ao lado de `user.go`
- **Mocks em subdiretório**: `mocks/user_repository_mock.go`
- **Constantes no topo**: Logo após imports
- **Ordem de métodos**: construtores → públicos → privados

### 5.3 Error Handling (RFC 7807)

Todos os erros HTTP devem seguir **RFC 7807 - Problem Details for HTTP APIs**.

```go
// internal/domain/errors/errors.go
package errors

import "errors"

var (
    // Business errors
    ErrUserNotFound       = errors.New("user not found")
    ErrEmailAlreadyExists = errors.New("email already exists")
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrUnauthorized       = errors.New("unauthorized")
    ErrForbidden          = errors.New("forbidden")

    // Domain errors
    ErrInvalidEmail = errors.New("invalid email format")
    ErrInvalidCPF   = errors.New("invalid CPF")
)

// ProblemType define tipos de problemas (URIs)
const (
    ProblemTypeValidation      = "https://api.avantpro.com/problems/validation-error"
    ProblemTypeNotFound        = "https://api.avantpro.com/problems/not-found"
    ProblemTypeConflict        = "https://api.avantpro.com/problems/conflict"
    ProblemTypeUnauthorized    = "https://api.avantpro.com/problems/unauthorized"
    ProblemTypeForbidden       = "https://api.avantpro.com/problems/forbidden"
    ProblemTypeInternal        = "https://api.avantpro.com/problems/internal-error"
    ProblemTypeBadRequest      = "https://api.avantpro.com/problems/bad-request"
    ProblemTypeEmailExists     = "https://api.avantpro.com/problems/email-already-exists"
)

type DomainError struct {
    Type    string
    Title   string
    Message string
    Err     error
}

func (e *DomainError) Error() string {
    if e.Err != nil {
        return e.Message + ": " + e.Err.Error()
    }
    return e.Message
}

func (e *DomainError) Unwrap() error {
    return e.Err
}
```

**Helper para criar respostas RFC 7807:**

```go
// internal/handlers/dto/error_response.go
package dto

import (
    "net/http"
)

// ErrorResponse segue RFC 7807 (Problem Details for HTTP APIs)
type ErrorResponse struct {
    Type     string                 `json:"type"`
    Title    string                 `json:"title"`
    Status   int                    `json:"status"`
    Detail   string                 `json:"detail,omitempty"`
    Instance string                 `json:"instance,omitempty"`
    Errors   []ValidationError      `json:"errors,omitempty"`
    Meta     map[string]interface{} `json:"meta,omitempty"`
}

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Tag     string `json:"tag,omitempty"`
}

// NewErrorResponse cria resposta de erro RFC 7807
func NewErrorResponse(problemType, title string, status int, detail, instance string) ErrorResponse {
    return ErrorResponse{
        Type:     problemType,
        Title:    title,
        Status:   status,
        Detail:   detail,
        Instance: instance,
    }
}

// Helpers pré-definidos com suporte a i18n
// Todos recebem *gin.Context para acessar o localizer

func ValidationErrorResponse(c *gin.Context, errors []ValidationError) ErrorResponse {
    return ErrorResponse{
        Type:     "https://api.avantpro.com/problems/validation-error",
        Title:    T(c, "error.validation.title"),
        Status:   http.StatusBadRequest,
        Detail:   T(c, "error.validation.detail"),
        Instance: c.Request.URL.Path,
        Errors:   errors,
    }
}

func NotFoundErrorResponse(c *gin.Context, resourceKey string) ErrorResponse {
    return ErrorResponse{
        Type:     "https://api.avantpro.com/problems/not-found",
        Title:    T(c, "error.not_found.title"),
        Status:   http.StatusNotFound,
        Detail:   T(c, "error.not_found.detail", map[string]interface{}{"Resource": resourceKey}),
        Instance: c.Request.URL.Path,
    }
}

func ConflictErrorResponse(c *gin.Context, detailKey string, data ...map[string]interface{}) ErrorResponse {
    var templateData map[string]interface{}
    if len(data) > 0 {
        templateData = data[0]
    }

    return ErrorResponse{
        Type:     "https://api.avantpro.com/problems/conflict",
        Title:    T(c, "error.conflict.title"),
        Status:   http.StatusConflict,
        Detail:   T(c, detailKey, templateData),
        Instance: c.Request.URL.Path,
    }
}

func UnauthorizedErrorResponse(c *gin.Context) ErrorResponse {
    return ErrorResponse{
        Type:     "https://api.avantpro.com/problems/unauthorized",
        Title:    T(c, "error.unauthorized.title"),
        Status:   http.StatusUnauthorized,
        Detail:   T(c, "error.unauthorized.detail"),
        Instance: c.Request.URL.Path,
    }
}

func ForbiddenErrorResponse(c *gin.Context) ErrorResponse {
    return ErrorResponse{
        Type:     "https://api.avantpro.com/problems/forbidden",
        Title:    T(c, "error.forbidden.title"),
        Status:   http.StatusForbidden,
        Detail:   T(c, "error.forbidden.detail"),
        Instance: c.Request.URL.Path,
    }
}

func InternalErrorResponse(c *gin.Context) ErrorResponse {
    return ErrorResponse{
        Type:     "https://api.avantpro.com/problems/internal-error",
        Title:    T(c, "error.internal.title"),
        Status:   http.StatusInternalServerError,
        Detail:   T(c, "error.internal.detail"),
        Instance: c.Request.URL.Path,
    }
}

// T é um helper para tradução (wrapper do i18n.T)
func T(c *gin.Context, messageID string, templateData ...map[string]interface{}) string {
    localizer, exists := c.Get("localizer")
    if !exists {
        return messageID // Fallback
    }

    var data map[string]interface{}
    if len(templateData) > 0 {
        data = templateData[0]
    }

    msg, err := localizer.(*i18n.Localizer).Localize(&i18n.LocalizeConfig{
        MessageID:    messageID,
        TemplateData: data,
    })

    if err != nil {
        return messageID
    }

    return msg
}
```

**Exemplo de resposta JSON (RFC 7807):**

```http
HTTP/1.1 400 Bad Request
Content-Type: application/problem+json

{
  "type": "https://api.avantpro.com/problems/validation-error",
  "title": "Validation Failed",
  "status": 400,
  "detail": "One or more fields failed validation",
  "instance": "/api/v1/users",
  "errors": [
    {
      "field": "email",
      "message": "must be a valid email address",
      "tag": "email"
    },
    {
      "field": "name",
      "message": "must be at least 2 characters",
      "tag": "min"
    }
  ]
}
```

**Middleware para Content-Type RFC 7807:**

```go
// internal/handlers/middleware/error_handler.go
package middleware

import (
    "github.com/gin-gonic/gin"
)

// RFC7807ContentType configura Content-Type correto para erros
func RFC7807ContentType() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        // Se resposta for erro (4xx ou 5xx), usar application/problem+json
        if c.Writer.Status() >= 400 {
            c.Header("Content-Type", "application/problem+json; charset=utf-8")
        }
    }
}
```

### 5.4 Context Usage

```go
// Sempre passar context como primeiro parâmetro
func (s *UserService) CreateUser(ctx context.Context, input CreateUserInput) error

// Usar context para:
// 1. Cancelamento
// 2. Timeout
// 3. Request-scoped values (user ID, trace ID)
// 4. Transações (injetar *gorm.DB no ctx)

// Exemplo: extrair user ID do contexto
func getUserID(ctx context.Context) string {
    if userID, ok := ctx.Value("user_id").(string); ok {
        return userID
    }
    return ""
}
```

---

## 6. Dependency Injection

### 6.1 Wire (Google)

Usar `google/wire` para DI automática:

```go
// cmd/api/wire.go
//go:build wireinject
// +build wireinject

package main

import (
    "github.com/google/wire"
    "avantpro-backend/internal/services"
    "avantpro-backend/internal/handlers/http"
    // ...
)

func InitializeApp() (*App, error) {
    wire.Build(
        // Infrastructure
        NewDatabase,
        NewLogger,

        // Repositories
        postgres.NewUserRepository,
        postgres.NewUnitOfWork,

        // Gateways
        email.NewSMTPGateway,

        // Services
        services.NewUserService,
        services.NewAuthService,

        // Handlers
        http.NewUserHandler,
        http.NewAuthHandler,

        // App
        NewApp,
    )
    return &App{}, nil
}
```

### 6.2 Main Application

```go
// cmd/api/main.go
package main

import (
    "context"
    "log"
    "os/signal"
    "syscall"
)

func main() {
    // Inicializar app com Wire
    app, err := InitializeApp()
    if err != nil {
        log.Fatal("failed to initialize app:", err)
    }

    // Context com signal handling
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // Executar app
    if err := app.Run(ctx); err != nil {
        log.Fatal("app failed:", err)
    }
}

type App struct {
    router *gin.Engine
    config *config.Config
    logger domain.Logger
}

func (a *App) Run(ctx context.Context) error {
    // Setup routes
    a.setupRoutes()

    // Start server
    server := &http.Server{
        Addr:    ":" + a.config.Port,
        Handler: a.router,
    }

    // Graceful shutdown
    go func() {
        <-ctx.Done()
        server.Shutdown(context.Background())
    }()

    a.logger.Info("server starting", "port", a.config.Port)
    return server.ListenAndServe()
}
```

---

## 7. Próximos Documentos

Este é o documento principal. Documentos complementares:

1. **01-autenticacao-autorizacao.md** - JWT, OAuth2, RBAC
2. **02-validacao-i18n.md** - Validação e internacionalização
3. **03-testes.md** - Estratégias de testes (unit, integration, e2e)
4. **04-migrations-database.md** - Migrations e boas práticas de banco
5. **05-deploy-devops.md** - Docker, CI/CD, deployment
6. **06-exemplos-praticos.md** - Exemplos completos de features

---

**Versão**: 2.0
**Última Atualização**: 05/11/2025
**Autor**: Rafael (com assistência Claude Code)
