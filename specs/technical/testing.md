# Guia de Testes

**Versão**: 2.0
**Data**: 05/11/2025

---

## 1. Visão Geral

Estratégia de testes em múltiplas camadas:
- **Unit Tests**: Testes isolados com mocks (testify/mock)
- **Integration Tests**: Testes com banco real (testcontainers)
- **E2E Tests**: Testes da API completa via HTTP
- **Table-Driven Tests**: Padrão idiomático Go para múltiplos casos

---

## 2. Pirâmide de Testes

```
        ▲
       ╱ ╲
      ╱ E2E╲          ~10% - Testes end-to-end
     ╱───────╲        Lentos, alto custo, alta confiança
    ╱         ╲
   ╱Integration╲     ~30% - Testes de integração
  ╱─────────────╲    Médios, banco real, interações
 ╱               ╲
╱   Unit  Tests  ╲   ~60% - Testes unitários
─────────────────── Rápidos, isolados, mocks
```

**Princípios:**
- Maioria dos testes deve ser unitária (rápida, isolada)
- Integration tests para validar integrações críticas
- E2E tests para fluxos principais do usuário
- Executar tests unitários em todo commit
- Integration/E2E em CI/CD

---

## 3. Setup de Testes

### 3.1 Dependências

```bash
go get -u github.com/stretchr/testify
go get -u github.com/testcontainers/testcontainers-go
go get -u github.com/testcontainers/testcontainers-go/modules/postgres
go get -u github.com/testcontainers/testcontainers-go/modules/redis
```

### 3.2 Estrutura de Diretórios

```
avantpro-backend/
├── internal/
│   ├── domain/
│   │   ├── entities/
│   │   │   ├── user.go
│   │   │   └── user_test.go          # Unit tests ao lado do código
│   │   └── valueobjects/
│   │       ├── email.go
│   │       └── email_test.go
│   │
│   ├── services/
│   │   ├── user_service.go
│   │   └── user_service_test.go      # Unit tests com mocks
│   │
│   └── handlers/
│       └── http/
│           ├── user_handler.go
│           └── user_handler_test.go  # Unit tests do handler
│
├── tests/
│   ├── integration/                   # Integration tests
│   │   ├── user_repository_test.go
│   │   ├── auth_service_test.go
│   │   └── testhelpers/
│   │       ├── database.go
│   │       └── fixtures.go
│   │
│   ├── e2e/                          # E2E tests
│   │   ├── user_api_test.go
│   │   ├── auth_api_test.go
│   │   └── setup_test.go
│   │
│   ├── mocks/                        # Mocks gerados
│   │   ├── user_repository_mock.go
│   │   ├── email_gateway_mock.go
│   │   └── generate.go
│   │
│   └── fixtures/                     # Dados de teste
│       ├── users.json
│       └── sql/
│           └── seed.sql
│
└── Makefile                          # Comandos de teste
```

---

## 4. Unit Tests

### 4.1 Testando Entities

```go
// internal/domain/entities/user_test.go
package entities_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/valueobjects"
)

func TestUser_IsAdmin(t *testing.T) {
    // Arrange
    adminEmail, _ := valueobjects.NewEmail("admin@example.com")
    admin := &entities.User{
        ID:    "1",
        Email: adminEmail,
        Role:  entities.RoleAdmin,
    }

    userEmail, _ := valueobjects.NewEmail("user@example.com")
    user := &entities.User{
        ID:    "2",
        Email: userEmail,
        Role:  entities.RoleUser,
    }

    // Act & Assert
    assert.True(t, admin.IsAdmin())
    assert.False(t, user.IsAdmin())
}

func TestUser_Validate(t *testing.T) {
    tests := []struct {
        name    string
        user    *entities.User
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid user",
            user: &entities.User{
                ID:    "1",
                Email: mustCreateEmail("valid@example.com"),
                Name:  "John Doe",
                Role:  entities.RoleUser,
            },
            wantErr: false,
        },
        {
            name: "empty name",
            user: &entities.User{
                ID:    "1",
                Email: mustCreateEmail("valid@example.com"),
                Name:  "",
                Role:  entities.RoleUser,
            },
            wantErr: true,
            errMsg:  "name is required",
        },
        {
            name: "name too short",
            user: &entities.User{
                ID:    "1",
                Email: mustCreateEmail("valid@example.com"),
                Name:  "J",
                Role:  entities.RoleUser,
            },
            wantErr: true,
            errMsg:  "name must be at least 2 characters",
        },
        {
            name: "invalid role",
            user: &entities.User{
                ID:    "1",
                Email: mustCreateEmail("valid@example.com"),
                Name:  "John Doe",
                Role:  entities.Role("invalid"),
            },
            wantErr: true,
            errMsg:  "invalid role",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.user.Validate()

            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}

func mustCreateEmail(email string) valueobjects.Email {
    e, _ := valueobjects.NewEmail(email)
    return e
}
```

### 4.2 Testando Value Objects

```go
// internal/domain/valueobjects/email_test.go
package valueobjects_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "avantpro-backend/internal/domain/valueobjects"
)

func TestNewEmail(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid email",
            input:   "user@example.com",
            want:    "user@example.com",
            wantErr: false,
        },
        {
            name:    "valid email with uppercase",
            input:   "User@Example.COM",
            want:    "user@example.com", // deve normalizar
            wantErr: false,
        },
        {
            name:    "email with spaces",
            input:   "  user@example.com  ",
            want:    "user@example.com", // deve trimmar
            wantErr: false,
        },
        {
            name:    "invalid email - missing @",
            input:   "userexample.com",
            wantErr: true,
        },
        {
            name:    "invalid email - missing domain",
            input:   "user@",
            wantErr: true,
        },
        {
            name:    "invalid email - missing local part",
            input:   "@example.com",
            wantErr: true,
        },
        {
            name:    "empty email",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            email, err := valueobjects.NewEmail(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, email.String())
            }
        })
    }
}
```

### 4.3 Testando Services com Mocks

```go
// tests/mocks/user_repository_mock.go
package mocks

import (
    "context"
    "github.com/stretchr/testify/mock"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/repositories"
)

type UserRepositoryMock struct {
    mock.Mock
}

func NewUserRepositoryMock() *UserRepositoryMock {
    return &UserRepositoryMock{}
}

func (m *UserRepositoryMock) Create(ctx context.Context, user *entities.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

func (m *UserRepositoryMock) FindByID(ctx context.Context, id string) (*entities.User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entities.User), args.Error(1)
}

func (m *UserRepositoryMock) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
    args := m.Called(ctx, email)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entities.User), args.Error(1)
}

func (m *UserRepositoryMock) Update(ctx context.Context, user *entities.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

func (m *UserRepositoryMock) Delete(ctx context.Context, id string) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}

func (m *UserRepositoryMock) List(ctx context.Context, filters repositories.UserFilters) ([]*entities.User, error) {
    args := m.Called(ctx, filters)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).([]*entities.User), args.Error(1)
}
```

```go
// internal/services/user_service_test.go
package services_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/valueobjects"
    "avantpro-backend/internal/services"
    "avantpro-backend/tests/mocks"
)

func TestUserService_CreateUser(t *testing.T) {
    // Setup
    userRepo := mocks.NewUserRepositoryMock()
    emailGateway := mocks.NewEmailGatewayMock()
    uow := mocks.NewUnitOfWorkMock()
    logger := mocks.NewLoggerMock()

    service := services.NewUserService(userRepo, emailGateway, uow, logger)

    t.Run("success - creates user and sends email", func(t *testing.T) {
        // Arrange
        input := services.CreateUserInput{
            Email: "newuser@example.com",
            Name:  "New User",
        }

        // Mock: email não existe
        userRepo.On("FindByEmail", mock.Anything, input.Email).Return(nil, nil).Once()

        // Mock: UnitOfWork executa função de transação
        uow.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
            Run(func(args mock.Arguments) {
                // Executar a função passada
                fn := args.Get(1).(func(context.Context) error)
                fn(context.Background())
            }).
            Return(nil).
            Once()

        // Mock: criar usuário
        userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.User")).
            Return(nil).
            Once()

        // Mock: enviar email
        emailGateway.On("SendWelcomeEmail", mock.Anything, input.Email, input.Name).
            Return(nil).
            Once()

        // Mock: logger
        logger.On("Info", mock.Anything, mock.Anything).Maybe()

        // Act
        user, err := service.CreateUser(context.Background(), input)

        // Assert
        require.NoError(t, err)
        assert.NotNil(t, user)
        assert.Equal(t, input.Email, user.Email.String())
        assert.Equal(t, input.Name, user.Name)

        // Verificar que todos os mocks foram chamados
        userRepo.AssertExpectations(t)
        emailGateway.AssertExpectations(t)
        uow.AssertExpectations(t)
    })

    t.Run("error - email already exists", func(t *testing.T) {
        // Arrange
        input := services.CreateUserInput{
            Email: "existing@example.com",
            Name:  "Existing User",
        }

        existingUser := &entities.User{
            ID:    "123",
            Email: mustCreateEmail(input.Email),
            Name:  input.Name,
        }

        // Mock: email já existe
        userRepo.On("FindByEmail", mock.Anything, input.Email).
            Return(existingUser, nil).
            Once()

        logger.On("Info", mock.Anything, mock.Anything).Maybe()

        // Act
        user, err := service.CreateUser(context.Background(), input)

        // Assert
        require.Error(t, err)
        assert.Nil(t, user)
        assert.Equal(t, services.ErrEmailAlreadyExists, err)

        userRepo.AssertExpectations(t)
    })
}

func mustCreateEmail(email string) valueobjects.Email {
    e, _ := valueobjects.NewEmail(email)
    return e
}
```

---

## 5. Integration Tests

### 5.1 Setup com Testcontainers

```go
// tests/integration/testhelpers/database.go
package testhelpers

import (
    "context"
    "fmt"
    "testing"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "time"
)

type TestDatabase struct {
    Container testcontainers.Container
    DB        *gorm.DB
}

func SetupTestDatabase(t *testing.T) *TestDatabase {
    ctx := context.Background()

    // Criar container PostgreSQL
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(60*time.Second)),
    )
    if err != nil {
        t.Fatalf("failed to start postgres container: %s", err)
    }

    // Cleanup ao final do teste
    t.Cleanup(func() {
        if err := postgresContainer.Terminate(ctx); err != nil {
            t.Fatalf("failed to terminate container: %s", err)
        }
    })

    // Conectar ao banco
    host, _ := postgresContainer.Host(ctx)
    port, _ := postgresContainer.MappedPort(ctx, "5432")

    dsn := fmt.Sprintf(
        "host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
        host, port.Port(),
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to connect to database: %s", err)
    }

    // Executar migrations
    runMigrations(t, db)

    return &TestDatabase{
        Container: postgresContainer,
        DB:        db,
    }
}

func runMigrations(t *testing.T, db *gorm.DB) {
    // Auto-migrate entities de teste
    err := db.AutoMigrate(
        &UserModel{},
        &SubscriptionModel{},
        &PaymentModel{},
    )
    if err != nil {
        t.Fatalf("failed to run migrations: %s", err)
    }
}

func (td *TestDatabase) Cleanup(t *testing.T) {
    // Limpar todas as tabelas
    tables := []string{"users", "subscriptions", "payments"}
    for _, table := range tables {
        if err := td.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
            t.Fatalf("failed to truncate table %s: %s", table, err)
        }
    }
}
```

### 5.2 Integration Test de Repository

```go
// tests/integration/user_repository_test.go
package integration_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/valueobjects"
    "avantpro-backend/internal/infrastructure/persistence/postgres"
    "avantpro-backend/tests/integration/testhelpers"
)

func TestUserRepository_Create(t *testing.T) {
    // Setup
    testDB := testhelpers.SetupTestDatabase(t)
    defer testDB.Cleanup(t)

    repo := postgres.NewUserRepository(testDB.DB)
    ctx := context.Background()

    t.Run("success - creates user", func(t *testing.T) {
        // Arrange
        email, _ := valueobjects.NewEmail("test@example.com")
        user := &entities.User{
            Email: email,
            Name:  "Test User",
            Role:  entities.RoleUser,
        }

        // Act
        err := repo.Create(ctx, user)

        // Assert
        require.NoError(t, err)
        assert.NotEmpty(t, user.ID) // ID deve ter sido gerado

        // Verificar no banco
        found, err := repo.FindByID(ctx, user.ID)
        require.NoError(t, err)
        assert.Equal(t, user.Email.String(), found.Email.String())
        assert.Equal(t, user.Name, found.Name)
    })

    t.Run("error - duplicate email", func(t *testing.T) {
        // Arrange
        email, _ := valueobjects.NewEmail("duplicate@example.com")
        user1 := &entities.User{
            Email: email,
            Name:  "User 1",
            Role:  entities.RoleUser,
        }
        user2 := &entities.User{
            Email: email,
            Name:  "User 2",
            Role:  entities.RoleUser,
        }

        // Act
        err1 := repo.Create(ctx, user1)
        err2 := repo.Create(ctx, user2)

        // Assert
        require.NoError(t, err1)
        require.Error(t, err2) // Deve falhar por email duplicado
    })
}

func TestUserRepository_FindByEmail(t *testing.T) {
    testDB := testhelpers.SetupTestDatabase(t)
    defer testDB.Cleanup(t)

    repo := postgres.NewUserRepository(testDB.DB)
    ctx := context.Background()

    t.Run("found", func(t *testing.T) {
        // Arrange - criar usuário
        email, _ := valueobjects.NewEmail("find@example.com")
        user := &entities.User{
            Email: email,
            Name:  "Find Me",
            Role:  entities.RoleUser,
        }
        repo.Create(ctx, user)

        // Act
        found, err := repo.FindByEmail(ctx, "find@example.com")

        // Assert
        require.NoError(t, err)
        require.NotNil(t, found)
        assert.Equal(t, user.ID, found.ID)
    })

    t.Run("not found", func(t *testing.T) {
        // Act
        found, err := repo.FindByEmail(ctx, "notexist@example.com")

        // Assert
        require.NoError(t, err)
        assert.Nil(t, found)
    })
}

func TestUserRepository_List(t *testing.T) {
    testDB := testhelpers.SetupTestDatabase(t)
    defer testDB.Cleanup(t)

    repo := postgres.NewUserRepository(testDB.DB)
    ctx := context.Background()

    // Arrange - criar vários usuários
    users := []*entities.User{
        {Email: mustCreateEmail("admin1@example.com"), Name: "Admin 1", Role: entities.RoleAdmin},
        {Email: mustCreateEmail("admin2@example.com"), Name: "Admin 2", Role: entities.RoleAdmin},
        {Email: mustCreateEmail("user1@example.com"), Name: "User 1", Role: entities.RoleUser},
        {Email: mustCreateEmail("user2@example.com"), Name: "User 2", Role: entities.RoleUser},
    }

    for _, user := range users {
        repo.Create(ctx, user)
    }

    t.Run("list all", func(t *testing.T) {
        // Act
        found, err := repo.List(ctx, repositories.UserFilters{
            Page:     1,
            PageSize: 10,
        })

        // Assert
        require.NoError(t, err)
        assert.Len(t, found, 4)
    })

    t.Run("filter by role", func(t *testing.T) {
        // Act
        adminRole := entities.RoleAdmin
        found, err := repo.List(ctx, repositories.UserFilters{
            Role:     &adminRole,
            Page:     1,
            PageSize: 10,
        })

        // Assert
        require.NoError(t, err)
        assert.Len(t, found, 2)
        for _, user := range found {
            assert.Equal(t, entities.RoleAdmin, user.Role)
        }
    })

    t.Run("pagination", func(t *testing.T) {
        // Act - primeira página
        page1, err := repo.List(ctx, repositories.UserFilters{
            Page:     1,
            PageSize: 2,
        })

        require.NoError(t, err)
        assert.Len(t, page1, 2)

        // Act - segunda página
        page2, err := repo.List(ctx, repositories.UserFilters{
            Page:     2,
            PageSize: 2,
        })

        require.NoError(t, err)
        assert.Len(t, page2, 2)

        // IDs diferentes
        assert.NotEqual(t, page1[0].ID, page2[0].ID)
    })
}

func mustCreateEmail(email string) valueobjects.Email {
    e, _ := valueobjects.NewEmail(email)
    return e
}
```

---

## 6. E2E Tests

### 6.1 Setup do Server de Teste

```go
// tests/e2e/setup_test.go
package e2e_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
    "avantpro-backend/tests/integration/testhelpers"
)

type TestServer struct {
    Server   *httptest.Server
    DB       *testhelpers.TestDatabase
    Router   *gin.Engine
    Client   *http.Client
}

func SetupTestServer(t *testing.T) *TestServer {
    // Setup database
    testDB := testhelpers.SetupTestDatabase(t)

    // Setup Gin in test mode
    gin.SetMode(gin.TestMode)
    router := gin.New()

    // Inicializar app completo (DI, routes, middlewares)
    app := initializeTestApp(testDB.DB)
    setupRoutes(router, app)

    // Create test server
    server := httptest.NewServer(router)

    t.Cleanup(func() {
        server.Close()
    })

    return &TestServer{
        Server: server,
        DB:     testDB,
        Router: router,
        Client: server.Client(),
    }
}

func (ts *TestServer) Cleanup(t *testing.T) {
    ts.DB.Cleanup(t)
}
```

### 6.2 E2E Test Example

```go
// tests/e2e/user_api_test.go
package e2e_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserAPI_CreateUser(t *testing.T) {
    server := SetupTestServer(t)
    defer server.Cleanup(t)

    t.Run("success - creates user", func(t *testing.T) {
        // Arrange
        payload := map[string]interface{}{
            "email":    "newuser@example.com",
            "name":     "New User",
            "password": "SecurePass123!",
        }
        body, _ := json.Marshal(payload)

        // Act
        resp, err := server.Client.Post(
            server.Server.URL+"/api/v1/users",
            "application/json",
            bytes.NewBuffer(body),
        )

        // Assert
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusCreated, resp.StatusCode)

        var response map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&response)

        data := response["data"].(map[string]interface{})
        assert.Equal(t, payload["email"], data["email"])
        assert.Equal(t, payload["name"], data["name"])
        assert.NotEmpty(t, data["id"])
    })

    t.Run("error - validation failure", func(t *testing.T) {
        // Arrange
        payload := map[string]interface{}{
            "email": "invalid-email",
            "name":  "J", // Too short
        }
        body, _ := json.Marshal(payload)

        // Act
        resp, err := server.Client.Post(
            server.Server.URL+"/api/v1/users",
            "application/json",
            bytes.NewBuffer(body),
        )

        // Assert
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

        var response map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&response)

        assert.Equal(t, "validation_error", response["error"])
        assert.NotNil(t, response["details"])
    })

    t.Run("error - duplicate email", func(t *testing.T) {
        // Arrange - criar primeiro usuário
        payload1 := map[string]interface{}{
            "email":    "duplicate@example.com",
            "name":     "User 1",
            "password": "SecurePass123!",
        }
        body1, _ := json.Marshal(payload1)
        server.Client.Post(
            server.Server.URL+"/api/v1/users",
            "application/json",
            bytes.NewBuffer(body1),
        )

        // Act - tentar criar com mesmo email
        payload2 := map[string]interface{}{
            "email":    "duplicate@example.com",
            "name":     "User 2",
            "password": "SecurePass123!",
        }
        body2, _ := json.Marshal(payload2)

        resp, err := server.Client.Post(
            server.Server.URL+"/api/v1/users",
            "application/json",
            bytes.NewBuffer(body2),
        )

        // Assert
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusConflict, resp.StatusCode)
    })
}

func TestUserAPI_Authentication(t *testing.T) {
    server := SetupTestServer(t)
    defer server.Cleanup(t)

    t.Run("protected endpoint requires auth", func(t *testing.T) {
        // Act - tentar acessar sem token
        resp, err := server.Client.Get(server.Server.URL + "/api/v1/me")

        // Assert
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })

    t.Run("protected endpoint with valid token", func(t *testing.T) {
        // Arrange - criar usuário e fazer login
        token := createUserAndLogin(t, server)

        // Act - acessar com token
        req, _ := http.NewRequest("GET", server.Server.URL+"/api/v1/me", nil)
        req.Header.Set("Authorization", "Bearer "+token)

        resp, err := server.Client.Do(req)

        // Assert
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusOK, resp.StatusCode)
    })
}

func createUserAndLogin(t *testing.T, server *TestServer) string {
    // Criar usuário
    payload := map[string]interface{}{
        "email":    "auth@example.com",
        "name":     "Auth User",
        "password": "SecurePass123!",
    }
    body, _ := json.Marshal(payload)
    server.Client.Post(
        server.Server.URL+"/api/v1/users",
        "application/json",
        bytes.NewBuffer(body),
    )

    // Login
    loginPayload := map[string]interface{}{
        "email":    "auth@example.com",
        "password": "SecurePass123!",
    }
    loginBody, _ := json.Marshal(loginPayload)

    resp, _ := server.Client.Post(
        server.Server.URL+"/api/v1/auth/login",
        "application/json",
        bytes.NewBuffer(loginBody),
    )
    defer resp.Body.Close()

    var response map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&response)

    return response["access_token"].(string)
}
```

---

## 7. Test Helpers e Fixtures

### 7.1 Fixtures

```go
// tests/fixtures/users.go
package fixtures

import (
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/valueobjects"
)

func CreateTestUser(email, name string, role entities.Role) *entities.User {
    emailVO, _ := valueobjects.NewEmail(email)
    return &entities.User{
        ID:    generateUUID(),
        Email: emailVO,
        Name:  name,
        Role:  role,
    }
}

func CreateTestAdmin() *entities.User {
    return CreateTestUser("admin@example.com", "Admin User", entities.RoleAdmin)
}

func CreateTestRegularUser() *entities.User {
    return CreateTestUser("user@example.com", "Regular User", entities.RoleUser)
}
```

---

## 8. Comandos Make

```makefile
# Makefile

.PHONY: test test-unit test-integration test-e2e test-coverage

# Todos os testes
test:
	go test ./... -v

# Apenas unit tests (rápidos)
test-unit:
	go test ./internal/... -v -short

# Integration tests
test-integration:
	go test ./tests/integration/... -v

# E2E tests
test-e2e:
	go test ./tests/e2e/... -v

# Coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Watch mode (com entr)
test-watch:
	find . -name '*.go' | entr -c make test/unit
```

---

## 9. Best Practices

### 9.1 Checklist

- ✅ AAA pattern: Arrange, Act, Assert
- ✅ Table-driven tests para múltiplos casos
- ✅ Nomes descritivos: `TestService_Method_Condition`
- ✅ t.Run para subtestes
- ✅ require para asserções críticas (para execução)
- ✅ assert para verificações não-críticas
- ✅ Mocks isolam unit tests
- ✅ Integration tests com banco real
- ✅ Cleanup após testes (t.Cleanup)
- ✅ Testcontainers para dependências externas
- ✅ Coverage > 80% para código crítico
- ✅ Tests rápidos (<1s unit, <10s integration)

### 9.2 Quando Mockar vs Real

| Cenário | Mock | Real |
|---------|------|------|
| **Unit test de Service** | ✅ Mock repository | ❌ |
| **Integration test de Repository** | ❌ | ✅ Testcontainers |
| **E2E test** | ❌ | ✅ Testcontainers |
| **External API (email, payment)** | ✅ Mock | ❌ |
| **Logger em tests** | ✅ Mock ou noop | ❌ |

---

**Versão**: 2.0
**Última Atualização**: 05/11/2025
