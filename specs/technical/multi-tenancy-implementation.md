# Implementação de Multi-Tenancy

**Versão**: 1.0
**Data**: 06/11/2025

---

## 1. Visão Geral

Este documento descreve a implementação técnica do sistema multi-tenant do AvantPro Backend usando o modelo **Shared Database + Shared Schema**.

**Modelo**: Row-Level Isolation com `organization_id`
**Localização**: `internal/infrastructure/persistence/`, `internal/services/`

**Specs Relacionadas**: `specs/functional/multi-tenancy.md`

---

## 2. Auth Service - Multi-Organization Login

### 2.1 Implementação Completa do Fluxo de Login

```go
// internal/services/auth_service.go
package services

import (
    "context"
    "errors"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/repositories"
    "avantpro-backend/internal/infrastructure/auth"
)

type AuthService struct {
    userRepo      repositories.UserRepository
    orgMemberRepo repositories.OrganizationMemberRepository
    jwtService    *auth.JWTService
    tokenStore    *auth.RefreshTokenStore
}

// Login com suporte a múltiplas organizations
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
    // 1. Validar credenciais
    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return nil, errors.New("invalid_credentials")
    }

    if !user.Password.Verify(password) {
        return nil, errors.New("invalid_credentials")
    }

    // Verificar status do usuário
    if user.Status != entities.UserStatusActive {
        return nil, errors.New("user_inactive")
    }

    // 2. Buscar organizations do usuário
    members, err := s.orgMemberRepo.FindByUserID(ctx, user.ID)
    if err != nil {
        return nil, err
    }

    if len(members) == 0 {
        return nil, errors.New("user_has_no_organizations")
    }

    // 3. Caso A: Usuário tem exatamente 1 organization
    if len(members) == 1 {
        return s.generateTokensForOrganization(ctx, user, members[0])
    }

    // 4. Caso B: Usuário tem múltiplas organizations
    return s.generateTempTokenWithOrganizations(user, members)
}

// generateTokensForOrganization gera JWT final para uma organization
func (s *AuthService) generateTokensForOrganization(
    ctx context.Context,
    user *entities.User,
    member *entities.OrganizationMember,
) (*LoginResponse, error) {
    // Obter permissões baseado na role
    permissions := s.getPermissionsForRole(member.Role)

    // Gerar access token
    accessToken, err := s.jwtService.GenerateAccessToken(
        user.ID,
        user.Email.Value(),
        member.OrganizationID,
        member.Organization.Name,
        string(member.Role),
        permissions,
    )
    if err != nil {
        return nil, err
    }

    // Gerar refresh token
    refreshToken, err := s.jwtService.GenerateRefreshToken(
        user.ID,
        member.OrganizationID,
    )
    if err != nil {
        return nil, err
    }

    // Armazenar refresh token no Redis (7 dias)
    jti := extractJTI(refreshToken)
    err = s.tokenStore.Store(ctx, jti, user.ID, auth.RefreshTokenDuration)
    if err != nil {
        return nil, err
    }

    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        Organization: &OrganizationInfo{
            ID:   member.OrganizationID,
            Name: member.Organization.Name,
            Role: string(member.Role),
        },
    }, nil
}

// generateTempTokenWithOrganizations gera token temporário + lista de orgs
func (s *AuthService) generateTempTokenWithOrganizations(
    user *entities.User,
    members []*entities.OrganizationMember,
) (*LoginResponse, error) {
    // Gerar token temporário (15 min, tipo: organization_selection)
    tempToken, err := s.jwtService.GenerateTempToken(user.ID, user.Email.Value())
    if err != nil {
        return nil, err
    }

    // Converter members para OrganizationInfo
    orgs := make([]OrganizationInfo, len(members))
    for i, member := range members {
        orgs[i] = OrganizationInfo{
            ID:   member.OrganizationID,
            Name: member.Organization.Name,
            Role: string(member.Role),
        }
    }

    return &LoginResponse{
        RequiresOrganizationSelection: true,
        TempToken:                      tempToken,
        Organizations:                  orgs,
    }, nil
}

// SelectOrganization - Usuário escolhe organization após login
func (s *AuthService) SelectOrganization(
    ctx context.Context,
    tempToken, organizationID string,
) (*LoginResponse, error) {
    // 1. Validar temp token
    claims, err := s.jwtService.ValidateToken(tempToken)
    if err != nil {
        return nil, errors.New("invalid_temp_token")
    }

    if claims.Type != auth.TokenTypeOrganizationSelection {
        return nil, errors.New("invalid_token_type")
    }

    // 2. Validar que user é membro da organization escolhida
    member, err := s.orgMemberRepo.FindByUserAndOrganization(
        ctx,
        claims.UserID,
        organizationID,
    )
    if err != nil || member == nil {
        return nil, errors.New("user_not_member_of_organization")
    }

    // 3. Buscar usuário
    user, err := s.userRepo.FindByID(ctx, claims.UserID)
    if err != nil {
        return nil, errors.New("user_not_found")
    }

    // 4. Gerar JWT final para organization escolhida
    return s.generateTokensForOrganization(ctx, user, member)
}

// SwitchOrganization - Trocar de organization (já autenticado)
func (s *AuthService) SwitchOrganization(
    ctx context.Context,
    userID, newOrganizationID string,
) (*LoginResponse, error) {
    // 1. Validar que user é membro da nova organization
    member, err := s.orgMemberRepo.FindByUserAndOrganization(
        ctx,
        userID,
        newOrganizationID,
    )
    if err != nil || member == nil {
        return nil, errors.New("user_not_member_of_organization")
    }

    // 2. Buscar usuário
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        return nil, errors.New("user_not_found")
    }

    // 3. Gerar novos tokens para nova organization
    return s.generateTokensForOrganization(ctx, user, member)
}

// getPermissionsForRole retorna permissões baseado na role
func (s *AuthService) getPermissionsForRole(role entities.Role) []string {
    // Ver specs/functional/auth.md para mapeamento completo
    switch role {
    case entities.RoleAdmin:
        return []string{"*:*"}
    case entities.RoleUser:
        return []string{
            "users.read",
            "subscriptions.*",
            "payments.read",
        }
    case entities.RoleGuest:
        return []string{}
    default:
        return []string{}
    }
}
```

---

## 3. Repository Pattern com Isolamento

### 3.1 Subscription Repository (Tabela com organization_id)

```go
// internal/infrastructure/persistence/postgres/subscription_repository.go
package postgres

import (
    "context"
    "errors"

    "gorm.io/gorm"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/repositories"
)

type SubscriptionRepository struct {
    db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) repositories.SubscriptionRepository {
    return &SubscriptionRepository{db: db}
}

// FindByID busca subscription por ID (COM filtro de organization)
func (r *SubscriptionRepository) FindByID(ctx context.Context, id string) (*entities.Subscription, error) {
    // Extrair organization_id do contexto (injetado pelo middleware)
    organizationID, ok := ctx.Value("organization_id").(string)
    if !ok {
        return nil, errors.New("organization_id not found in context")
    }

    var model SubscriptionModel

    // Query SEMPRE filtra por organization_id - ISOLAMENTO
    err := r.db.WithContext(ctx).
        Where("id = ? AND organization_id = ? AND deleted_at IS NULL", id, organizationID).
        First(&model).
        Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("subscription_not_found")
        }
        return nil, err
    }

    return toSubscriptionEntity(&model), nil
}

// List lista subscriptions (COM filtro de organization)
func (r *SubscriptionRepository) List(ctx context.Context) ([]*entities.Subscription, error) {
    organizationID, ok := ctx.Value("organization_id").(string)
    if !ok {
        return nil, errors.New("organization_id not found in context")
    }

    var models []SubscriptionModel

    // Query SEMPRE filtra por organization_id
    err := r.db.WithContext(ctx).
        Where("organization_id = ? AND deleted_at IS NULL", organizationID).
        Order("created_at DESC").
        Find(&models).
        Error

    if err != nil {
        return nil, err
    }

    subscriptions := make([]*entities.Subscription, len(models))
    for i, model := range models {
        subscriptions[i] = toSubscriptionEntity(&model)
    }

    return subscriptions, nil
}

// Create cria subscription (COM organization_id do contexto)
func (r *SubscriptionRepository) Create(ctx context.Context, subscription *entities.Subscription) error {
    organizationID, ok := ctx.Value("organization_id").(string)
    if !ok {
        return errors.New("organization_id not found in context")
    }

    model := toSubscriptionModel(subscription)
    model.OrganizationID = organizationID  // FORÇA organization_id do contexto

    return r.db.WithContext(ctx).Create(model).Error
}

// Update atualiza subscription (COM validação de organization)
func (r *SubscriptionRepository) Update(ctx context.Context, subscription *entities.Subscription) error {
    organizationID, ok := ctx.Value("organization_id").(string)
    if !ok {
        return errors.New("organization_id not found in context")
    }

    model := toSubscriptionModel(subscription)

    // Update APENAS se pertence à organization do contexto
    result := r.db.WithContext(ctx).
        Where("id = ? AND organization_id = ? AND deleted_at IS NULL", model.ID, organizationID).
        Updates(model)

    if result.Error != nil {
        return result.Error
    }

    if result.RowsAffected == 0 {
        return errors.New("subscription_not_found")
    }

    return nil
}

// Delete soft delete (COM validação de organization)
func (r *SubscriptionRepository) Delete(ctx context.Context, id string) error {
    organizationID, ok := ctx.Value("organization_id").(string)
    if !ok {
        return errors.New("organization_id not found in context")
    }

    // Soft delete APENAS se pertence à organization
    result := r.db.WithContext(ctx).
        Model(&SubscriptionModel{}).
        Where("id = ? AND organization_id = ? AND deleted_at IS NULL", id, organizationID).
        Update("deleted_at", time.Now().Unix())

    if result.Error != nil {
        return result.Error
    }

    if result.RowsAffected == 0 {
        return errors.New("subscription_not_found")
    }

    return nil
}
```

### 3.2 User Repository (Tabela GLOBAL - sem organization_id)

```go
// internal/infrastructure/persistence/postgres/user_repository.go
package postgres

import (
    "context"
    "errors"

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

// FindByEmail busca user por email (SEM filtro de organization)
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
    var model UserModel

    // Query NÃO filtra por organization (tabela global)
    err := r.db.WithContext(ctx).
        Where("email = ? AND deleted_at IS NULL", email).
        First(&model).
        Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("user_not_found")
        }
        return nil, err
    }

    // Carregar UserAccount (1:1)
    var accountModel UserAccountModel
    r.db.WithContext(ctx).
        Where("user_id = ? AND deleted_at IS NULL", model.ID).
        First(&accountModel)

    return toUserEntity(&model, &accountModel), nil
}

// FindByID busca user por ID (SEM filtro de organization)
func (r *UserRepository) FindByID(ctx context.Context, id string) (*entities.User, error) {
    var model UserModel

    err := r.db.WithContext(ctx).
        Where("id = ? AND deleted_at IS NULL", id).
        First(&model).
        Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("user_not_found")
        }
        return nil, err
    }

    var accountModel UserAccountModel
    r.db.WithContext(ctx).
        Where("user_id = ? AND deleted_at IS NULL", model.ID).
        First(&accountModel)

    return toUserEntity(&model, &accountModel), nil
}

// Create cria user (SEM organization_id - tabela global)
func (r *UserRepository) Create(ctx context.Context, user *entities.User) error {
    userModel := toUserModel(user)
    accountModel := toUserAccountModel(user)

    // Transaction para criar User + UserAccount
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Create(userModel).Error; err != nil {
            return err
        }

        accountModel.UserID = userModel.ID
        if err := tx.Create(accountModel).Error; err != nil {
            return err
        }

        return nil
    })
}
```

### 3.3 OrganizationMember Repository (Cross-Organization Queries)

```go
// internal/infrastructure/persistence/postgres/organization_member_repository.go
package postgres

import (
    "context"

    "gorm.io/gorm"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/repositories"
)

type OrganizationMemberRepository struct {
    db *gorm.DB
}

func NewOrganizationMemberRepository(db *gorm.DB) repositories.OrganizationMemberRepository {
    return &OrganizationMemberRepository{db: db}
}

// FindByUserID retorna TODAS as organizations de um user
// Query cross-organization (SEM filtro de organization_id)
func (r *OrganizationMemberRepository) FindByUserID(
    ctx context.Context,
    userID string,
) ([]*entities.OrganizationMember, error) {
    var models []OrganizationMemberModel

    err := r.db.WithContext(ctx).
        Where("user_id = ? AND deleted_at IS NULL", userID).
        Preload("Organization"). // Eager load organization data
        Find(&models).
        Error

    if err != nil {
        return nil, err
    }

    members := make([]*entities.OrganizationMember, len(models))
    for i, model := range models {
        members[i] = toOrganizationMemberEntity(&model)
    }

    return members, nil
}

// FindByUserAndOrganization valida se user é membro de organization
func (r *OrganizationMemberRepository) FindByUserAndOrganization(
    ctx context.Context,
    userID, organizationID string,
) (*entities.OrganizationMember, error) {
    var model OrganizationMemberModel

    err := r.db.WithContext(ctx).
        Where("user_id = ? AND organization_id = ? AND deleted_at IS NULL", userID, organizationID).
        Preload("Organization").
        First(&model).
        Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil  // Não é membro
        }
        return nil, err
    }

    return toOrganizationMemberEntity(&model), nil
}

// FindByOrganization lista membros de uma organization
func (r *OrganizationMemberRepository) FindByOrganization(
    ctx context.Context,
    organizationID string,
) ([]*entities.OrganizationMember, error) {
    var models []OrganizationMemberModel

    err := r.db.WithContext(ctx).
        Where("organization_id = ? AND deleted_at IS NULL", organizationID).
        Preload("User").
        Find(&models).
        Error

    if err != nil {
        return nil, err
    }

    members := make([]*entities.OrganizationMember, len(models))
    for i, model := range models {
        members[i] = toOrganizationMemberEntity(&model)
    }

    return members, nil
}
```

---

## 4. Middleware de Organization

### 4.1 Organization Context Middleware

```go
// internal/handlers/http/middleware/organization_middleware.go
package middleware

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"
)

// OrganizationFromJWT extrai organization_id do JWT e valida membership
func OrganizationFromJWT() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Extrair organization_id do JWT (injetado pelo AuthMiddleware)
        organizationID, exists := c.Request.Context().Value("organization_id").(string)
        if !exists || organizationID == "" {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "missing_organization_id",
            })
            c.Abort()
            return
        }

        // 2. Extrair user_id do JWT
        userID, exists := c.Request.Context().Value("user_id").(string)
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "missing_user_id",
            })
            c.Abort()
            return
        }

        // 3. Adicionar organization_id ao contexto para uso nos repositories
        ctx := c.Request.Context()
        ctx = context.WithValue(ctx, "organization_id", organizationID)
        ctx = context.WithValue(ctx, "user_id", userID)

        c.Request = c.Request.WithContext(ctx)

        c.Next()
    }
}
```

---

## 5. Models GORM

### 5.1 Organization Model

```go
// internal/infrastructure/persistence/postgres/models.go
package postgres

type OrganizationModel struct {
    ID        string `gorm:"type:uuid;primary_key"`
    Name      string `gorm:"type:varchar(255);not null"`
    Status    string `gorm:"type:varchar(50);not null"`
    CreatedAt int64  `gorm:"autoCreateTime:milli"`
    UpdatedAt int64  `gorm:"autoUpdateTime:milli"`
    DeletedAt *int64 `gorm:"index"`
}

func (OrganizationModel) TableName() string {
    return "organizations"
}
```

### 5.2 OrganizationMember Model

```go
type OrganizationMemberModel struct {
    ID             string `gorm:"type:uuid;primary_key"`
    OrganizationID string `gorm:"type:uuid;not null;index"`
    UserID         string `gorm:"type:uuid;not null;index"`
    Role           string `gorm:"type:varchar(50);not null"`
    InvitedBy      string `gorm:"type:uuid"`
    InvitedAt      int64  `gorm:"not null"`
    JoinedAt       *int64
    CreatedAt      int64  `gorm:"autoCreateTime:milli"`
    DeletedAt      *int64 `gorm:"index"`

    // Eager loading relationships
    Organization *OrganizationModel `gorm:"foreignKey:OrganizationID"`
    User         *UserModel         `gorm:"foreignKey:UserID"`
}

func (OrganizationMemberModel) TableName() string {
    return "organization_members"
}
```

### 5.3 Subscription Model (Tabela de Negócio)

```go
type SubscriptionModel struct {
    ID             string `gorm:"type:uuid;primary_key"`
    OrganizationID string `gorm:"type:uuid;not null;index"` // ISOLAMENTO
    Name           string `gorm:"type:varchar(255);not null"`
    Price          float64 `gorm:"type:decimal(10,2);not null"`
    Status         string `gorm:"type:varchar(50);not null"`
    CreatedAt      int64  `gorm:"autoCreateTime:milli"`
    UpdatedAt      int64  `gorm:"autoUpdateTime:milli"`
    DeletedAt      *int64 `gorm:"index"`
}

func (SubscriptionModel) TableName() string {
    return "subscriptions"
}
```

---

## 6. Frontend Integration

### 6.1 JavaScript/TypeScript

```typescript
// services/auth.service.ts

interface LoginResponse {
  access_token?: string;
  refresh_token?: string;
  requires_organization_selection?: boolean;
  temp_token?: string;
  organizations?: Organization[];
  organization?: Organization;
}

interface Organization {
  id: string;
  name: string;
  role: string;
}

class AuthService {
  async login(email: string, password: string): Promise<LoginResponse> {
    const response = await api.post('/auth/login', { email, password });

    if (response.requires_organization_selection) {
      // Caso B: Múltiplas organizations
      this.storeTempToken(response.temp_token);
      this.showOrganizationSelector(response.organizations);
    } else {
      // Caso A: 1 organization
      this.storeTokens(response.access_token, response.refresh_token);
      this.setCurrentOrganization(response.organization);
      this.redirectToDashboard();
    }

    return response;
  }

  async selectOrganization(organizationId: string, tempToken: string): Promise<void> {
    const response = await api.post(
      '/auth/select-organization',
      { organization_id: organizationId },
      { headers: { Authorization: `Bearer ${tempToken}` } }
    );

    this.storeTokens(response.access_token, response.refresh_token);
    this.setCurrentOrganization(response.organization);
    this.redirectToDashboard();
  }

  async switchOrganization(organizationId: string): Promise<void> {
    const response = await api.post(
      '/auth/switch-organization',
      { organization_id: organizationId }
    );

    this.storeTokens(response.access_token, response.refresh_token);
    this.setCurrentOrganization(response.organization);

    // Recarregar dados da nova organization
    window.location.reload();
  }

  private storeTokens(accessToken: string, refreshToken: string): void {
    // Access token: memória ou sessionStorage (NÃO localStorage por segurança)
    sessionStorage.setItem('access_token', accessToken);

    // Refresh token: httpOnly cookie (ideal) ou localStorage
    localStorage.setItem('refresh_token', refreshToken);
  }

  private setCurrentOrganization(org: Organization): void {
    localStorage.setItem('current_organization', JSON.stringify(org));
  }
}
```

### 6.2 Organization Selector Component

```tsx
// components/OrganizationSelector.tsx

interface Props {
  organizations: Organization[];
  tempToken: string;
  onSelect: (orgId: string, token: string) => Promise<void>;
}

export function OrganizationSelector({ organizations, tempToken, onSelect }: Props) {
  return (
    <div className="organization-selector">
      <h2>Selecione a Organização</h2>
      <p>Você tem acesso a múltiplas organizações. Escolha uma para continuar:</p>

      <ul>
        {organizations.map(org => (
          <li key={org.id}>
            <button onClick={() => onSelect(org.id, tempToken)}>
              <div>
                <h3>{org.name}</h3>
                <span className="badge">{org.role}</span>
              </div>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

---

## 7. Testes de Isolamento

### 7.1 Teste de Isolamento entre Organizations

```go
// internal/infrastructure/persistence/postgres/subscription_repository_test.go
package postgres_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "avantpro-backend/internal/domain/entities"
)

func TestSubscriptionRepository_Isolation(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    repo := NewSubscriptionRepository(db)

    // Criar 2 organizations
    org1 := createTestOrganization(t, db, "Organization A")
    org2 := createTestOrganization(t, db, "Organization B")

    // Criar subscriptions em cada organization
    sub1 := createTestSubscription(t, db, org1.ID, "Sub A")
    sub2 := createTestSubscription(t, db, org2.ID, "Sub B")

    // Test: Buscar subscription da Organization A
    ctx := context.WithValue(context.Background(), "organization_id", org1.ID)

    result, err := repo.FindByID(ctx, sub1.ID)
    assert.NoError(t, err)
    assert.Equal(t, sub1.ID, result.ID)

    // Test: Tentar buscar subscription da Organization B usando contexto da A
    result, err = repo.FindByID(ctx, sub2.ID)
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Equal(t, "subscription_not_found", err.Error())
}

func TestSubscriptionRepository_List_OnlyOwnOrganization(t *testing.T) {
    db := setupTestDB(t)
    repo := NewSubscriptionRepository(db)

    org1 := createTestOrganization(t, db, "Organization A")
    org2 := createTestOrganization(t, db, "Organization B")

    // Criar 3 subscriptions na org1 e 2 na org2
    createTestSubscription(t, db, org1.ID, "Sub A1")
    createTestSubscription(t, db, org1.ID, "Sub A2")
    createTestSubscription(t, db, org1.ID, "Sub A3")
    createTestSubscription(t, db, org2.ID, "Sub B1")
    createTestSubscription(t, db, org2.ID, "Sub B2")

    // Listar subscriptions da Organization A
    ctx := context.WithValue(context.Background(), "organization_id", org1.ID)
    results, err := repo.List(ctx)

    assert.NoError(t, err)
    assert.Len(t, results, 3)  // Apenas as 3 da org A

    // Listar subscriptions da Organization B
    ctx = context.WithValue(context.Background(), "organization_id", org2.ID)
    results, err = repo.List(ctx)

    assert.NoError(t, err)
    assert.Len(t, results, 2)  // Apenas as 2 da org B
}
```

### 7.2 Teste de Multi-Organization User

```go
func TestAuthService_Login_MultipleOrganizations(t *testing.T) {
    db := setupTestDB(t)
    authService := setupAuthService(t, db)

    // Criar usuário
    user := createTestUser(t, db, "user@example.com", "Password123")

    // Criar 3 organizations
    orgA := createTestOrganization(t, db, "Organization A")
    orgB := createTestOrganization(t, db, "Organization B")
    orgC := createTestOrganization(t, db, "Organization C")

    // Adicionar user às 3 organizations com roles diferentes
    createTestMember(t, db, orgA.ID, user.ID, "admin")
    createTestMember(t, db, orgB.ID, user.ID, "user")
    createTestMember(t, db, orgC.ID, user.ID, "guest")

    // Login
    response, err := authService.Login(context.Background(), "user@example.com", "Password123")

    assert.NoError(t, err)
    assert.True(t, response.RequiresOrganizationSelection)
    assert.NotEmpty(t, response.TempToken)
    assert.Len(t, response.Organizations, 3)

    // Verificar roles corretas
    assert.Equal(t, "admin", response.Organizations[0].Role)
    assert.Equal(t, "user", response.Organizations[1].Role)
    assert.Equal(t, "guest", response.Organizations[2].Role)
}
```

---

## 8. Migrations

### 8.1 Migration de Organizations

```sql
-- internal/infrastructure/persistence/migrations/000003_create_organizations.up.sql

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

CREATE INDEX idx_organizations_status ON organizations(status);
CREATE INDEX idx_organizations_deleted_at ON organizations(deleted_at);
```

### 8.2 Migration de OrganizationMembers

```sql
-- internal/infrastructure/persistence/migrations/000004_create_organization_members.up.sql

CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    invited_by UUID REFERENCES users(id),
    invited_at BIGINT NOT NULL,
    joined_at BIGINT,
    created_at BIGINT NOT NULL,
    deleted_at BIGINT,
    UNIQUE(organization_id, user_id)
);

CREATE INDEX idx_organization_members_org ON organization_members(organization_id);
CREATE INDEX idx_organization_members_user ON organization_members(user_id);
CREATE INDEX idx_organization_members_deleted_at ON organization_members(deleted_at);
```

### 8.3 Migration de Subscriptions (Exemplo de Tabela de Negócio)

```sql
-- internal/infrastructure/persistence/migrations/000005_create_subscriptions.up.sql

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

-- CRITICAL: Índice em organization_id para performance de queries filtradas
CREATE INDEX idx_subscriptions_organization_id ON subscriptions(organization_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_deleted_at ON subscriptions(deleted_at);

-- Índice composto para queries comuns
CREATE INDEX idx_subscriptions_org_status ON subscriptions(organization_id, status)
    WHERE deleted_at IS NULL;
```

---

## 9. Segurança e Auditoria

### 9.1 Checklist de Segurança Multi-Tenant

- ✅ **Middleware SEMPRE valida JWT** e extrai organization_id
- ✅ **Contexto SEMPRE contém organization_id** e user_id
- ✅ **Repositories SEMPRE filtram** por organization_id (exceto tabelas globais)
- ✅ **Validar membership** antes de gerar JWT com organization_id
- ✅ **Índices em organization_id** para performance
- ✅ **Testes automatizados** validam isolamento
- ✅ **Code review** verifica queries sem filtro

### 9.2 Auditoria de Acesso

```go
// internal/infrastructure/logging/audit_logger.go
package logging

type AuditLog struct {
    Timestamp      time.Time `json:"timestamp"`
    UserID         string    `json:"user_id"`
    OrganizationID string    `json:"organization_id"`
    Action         string    `json:"action"`
    Resource       string    `json:"resource"`
    ResourceID     string    `json:"resource_id"`
    IPAddress      string    `json:"ip_address"`
    Success        bool      `json:"success"`
}

func LogAccess(ctx context.Context, action, resource, resourceID string, success bool) {
    log := AuditLog{
        Timestamp:      time.Now(),
        UserID:         ctx.Value("user_id").(string),
        OrganizationID: ctx.Value("organization_id").(string),
        Action:         action,
        Resource:       resource,
        ResourceID:     resourceID,
        Success:        success,
    }

    // Log estruturado (JSON)
    logger.Info("audit", "log", log)
}
```

---

## 10. Referências

**Specs Relacionadas**:
- `specs/functional/multi-tenancy.md` - Requisitos funcionais
- `specs/functional/auth.md` - Autenticação multi-organization
- `specs/technical/auth-implementation.md` - Implementação JWT

**Padrões**:
- Multi-Tenancy Shared Database Pattern
- Repository Pattern com Context
- Dependency Injection
