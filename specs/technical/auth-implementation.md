# Implementação de Autenticação - JWT e OAuth2

**Versão**: 1.0
**Data**: 06/11/2025

---

## 1. Visão Geral

Este documento descreve a implementação técnica do sistema de autenticação do AvantPro Backend usando:
- **JWT** (JSON Web Tokens) para autenticação stateless
- **OAuth2/OIDC** para login social (Google, GitHub)
- **Refresh Tokens** armazenados em Redis

**Localização**: `internal/infrastructure/auth/`

**Specs Relacionadas**: `specs/functional/auth.md`

---

## 2. JWT - JSON Web Tokens

### 2.1 Estrutura de Tokens

```go
// internal/infrastructure/auth/jwt.go
package auth

import (
    "time"
    "errors"
    "github.com/golang-jwt/jwt/v5"
)

const (
    AccessTokenDuration  = 15 * time.Minute
    RefreshTokenDuration = 7 * 24 * time.Hour
)

type TokenType string

const (
    TokenTypeAccess              TokenType = "access"
    TokenTypeRefresh             TokenType = "refresh"
    TokenTypeOrganizationSelection TokenType = "organization_selection"
)

// Claims customizados do JWT
type Claims struct {
    UserID           string   `json:"sub"`
    Email            string   `json:"email"`
    OrganizationID   string   `json:"organization_id,omitempty"`
    OrganizationName string   `json:"organization_name,omitempty"`
    Role             string   `json:"role,omitempty"`
    Permissions      []string `json:"permissions,omitempty"`
    Type             TokenType `json:"type"`
    jwt.RegisteredClaims
}

// JWTService gerencia geração e validação de tokens
type JWTService struct {
    secretKey     []byte
    issuer        string
    audience      string
}

func NewJWTService(secretKey, issuer, audience string) *JWTService {
    return &JWTService{
        secretKey: []byte(secretKey),
        issuer:    issuer,
        audience:  audience,
    }
}
```

### 2.2 Geração de Tokens

```go
// GenerateAccessToken gera JWT de acesso (15 min)
func (s *JWTService) GenerateAccessToken(
    userID, email, organizationID, organizationName, role string,
    permissions []string,
) (string, error) {
    now := time.Now()

    claims := Claims{
        UserID:           userID,
        Email:            email,
        OrganizationID:   organizationID,
        OrganizationName: organizationName,
        Role:             role,
        Permissions:      permissions,
        Type:             TokenTypeAccess,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.issuer,
            Audience:  jwt.ClaimStrings{s.audience},
            Subject:   userID,
            ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
            IssuedAt:  jwt.NewNumericDate(now),
            NotBefore: jwt.NewNumericDate(now),
            ID:        generateJTI(), // Unique token ID
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.secretKey)
}

// GenerateRefreshToken gera JWT de refresh (7 dias)
func (s *JWTService) GenerateRefreshToken(userID, organizationID string) (string, error) {
    now := time.Now()

    claims := Claims{
        UserID:         userID,
        OrganizationID: organizationID,
        Type:           TokenTypeRefresh,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.issuer,
            Subject:   userID,
            ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenDuration)),
            IssuedAt:  jwt.NewNumericDate(now),
            ID:        generateJTI(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.secretKey)
}

// GenerateTempToken gera token temporário para seleção de organization (15 min)
func (s *JWTService) GenerateTempToken(userID, email string) (string, error) {
    now := time.Now()

    claims := Claims{
        UserID: userID,
        Email:  email,
        Type:   TokenTypeOrganizationSelection,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.issuer,
            Subject:   userID,
            ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(now),
            ID:        generateJTI(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.secretKey)
}

// generateJTI gera ID único para o token (previne replay attacks)
func generateJTI() string {
    return uuid.New().String()
}
```

### 2.3 Validação de Tokens

```go
// ValidateToken valida e parseia um JWT
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Verificar algoritmo de assinatura
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("invalid signing method")
        }
        return s.secretKey, nil
    })

    if err != nil {
        return nil, err
    }

    if !token.Valid {
        return nil, errors.New("invalid token")
    }

    claims, ok := token.Claims.(*Claims)
    if !ok {
        return nil, errors.New("invalid claims")
    }

    return claims, nil
}

// ValidateAccessToken valida especificamente access tokens
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
    claims, err := s.ValidateToken(tokenString)
    if err != nil {
        return nil, err
    }

    if claims.Type != TokenTypeAccess {
        return nil, errors.New("token is not an access token")
    }

    return claims, nil
}

// ValidateRefreshToken valida especificamente refresh tokens
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
    claims, err := s.ValidateToken(tokenString)
    if err != nil {
        return nil, err
    }

    if claims.Type != TokenTypeRefresh {
        return nil, errors.New("token is not a refresh token")
    }

    return claims, nil
}
```

---

## 3. Refresh Token Storage (Redis)

### 3.1 Implementação

```go
// internal/infrastructure/auth/refresh_token_store.go
package auth

import (
    "context"
    "fmt"
    "time"
    "github.com/redis/go-redis/v9"
)

type RefreshTokenStore struct {
    client *redis.Client
}

func NewRefreshTokenStore(client *redis.Client) *RefreshTokenStore {
    return &RefreshTokenStore{client: client}
}

// Store armazena refresh token no Redis
func (s *RefreshTokenStore) Store(ctx context.Context, tokenID, userID string, ttl time.Duration) error {
    key := fmt.Sprintf("refresh_token:%s", tokenID)

    err := s.client.Set(ctx, key, userID, ttl).Err()
    if err != nil {
        return fmt.Errorf("failed to store refresh token: %w", err)
    }

    return nil
}

// Exists verifica se refresh token existe e é válido
func (s *RefreshTokenStore) Exists(ctx context.Context, tokenID string) (bool, error) {
    key := fmt.Sprintf("refresh_token:%s", tokenID)

    result, err := s.client.Exists(ctx, key).Result()
    if err != nil {
        return false, err
    }

    return result > 0, nil
}

// Revoke invalida um refresh token
func (s *RefreshTokenStore) Revoke(ctx context.Context, tokenID string) error {
    key := fmt.Sprintf("refresh_token:%s", tokenID)

    err := s.client.Del(ctx, key).Err()
    if err != nil {
        return fmt.Errorf("failed to revoke refresh token: %w", err)
    }

    return nil
}

// RevokeAllForUser invalida todos os refresh tokens de um usuário
func (s *RefreshTokenStore) RevokeAllForUser(ctx context.Context, userID string) error {
    pattern := fmt.Sprintf("refresh_token:*")

    var cursor uint64
    for {
        keys, nextCursor, err := s.client.Scan(ctx, cursor, pattern, 100).Result()
        if err != nil {
            return err
        }

        for _, key := range keys {
            storedUserID, err := s.client.Get(ctx, key).Result()
            if err != nil {
                continue
            }

            if storedUserID == userID {
                s.client.Del(ctx, key)
            }
        }

        cursor = nextCursor
        if cursor == 0 {
            break
        }
    }

    return nil
}
```

---

## 4. Auth Service

### 4.1 Service de Autenticação

```go
// internal/services/auth_service.go
package services

import (
    "context"
    "errors"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/repositories"
    "avantpro-backend/internal/domain/valueobjects"
    "avantpro-backend/internal/infrastructure/auth"
)

type AuthService struct {
    userRepo       repositories.UserRepository
    orgMemberRepo  repositories.OrganizationMemberRepository
    jwtService     *auth.JWTService
    tokenStore     *auth.RefreshTokenStore
}

func NewAuthService(
    userRepo repositories.UserRepository,
    orgMemberRepo repositories.OrganizationMemberRepository,
    jwtService *auth.JWTService,
    tokenStore *auth.RefreshTokenStore,
) *AuthService {
    return &AuthService{
        userRepo:      userRepo,
        orgMemberRepo: orgMemberRepo,
        jwtService:    jwtService,
        tokenStore:    tokenStore,
    }
}

// LoginResponse representa resposta de login
type LoginResponse struct {
    AccessToken                   string                   `json:"access_token,omitempty"`
    RefreshToken                  string                   `json:"refresh_token,omitempty"`
    RequiresOrganizationSelection bool                     `json:"requires_organization_selection,omitempty"`
    TempToken                     string                   `json:"temp_token,omitempty"`
    Organizations                 []OrganizationInfo       `json:"organizations,omitempty"`
    Organization                  *OrganizationInfo        `json:"organization,omitempty"`
}

type OrganizationInfo struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Role string `json:"role"`
}

// Login autentica usuário e retorna tokens
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
    // 1. Validar credenciais
    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return nil, errors.New("invalid_credentials")
    }

    if !user.Password.Verify(password) {
        return nil, errors.New("invalid_credentials")
    }

    // Verificar se usuário está ativo
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

    // 3. Caso 1 organization: gerar JWT final direto
    if len(members) == 1 {
        member := members[0]
        permissions := s.getPermissionsForRole(member.Role)

        accessToken, _ := s.jwtService.GenerateAccessToken(
            user.ID,
            user.Email.Value(),
            member.OrganizationID,
            member.Organization.Name,
            string(member.Role),
            permissions,
        )

        refreshToken, _ := s.jwtService.GenerateRefreshToken(user.ID, member.OrganizationID)

        // Armazenar refresh token no Redis
        s.tokenStore.Store(ctx, extractJTI(refreshToken), user.ID, auth.RefreshTokenDuration)

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

    // 4. Múltiplas organizations: gerar token temporário
    tempToken, _ := s.jwtService.GenerateTempToken(user.ID, user.Email.Value())

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

// RefreshAccessToken renova access token usando refresh token
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
    // 1. Validar refresh token
    claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
    if err != nil {
        return "", errors.New("invalid_refresh_token")
    }

    // 2. Verificar se existe no Redis
    exists, err := s.tokenStore.Exists(ctx, claims.ID)
    if err != nil || !exists {
        return "", errors.New("invalid_refresh_token")
    }

    // 3. Buscar usuário e organization member
    user, err := s.userRepo.FindByID(ctx, claims.UserID)
    if err != nil {
        return "", errors.New("user_not_found")
    }

    member, err := s.orgMemberRepo.FindByUserAndOrganization(ctx, claims.UserID, claims.OrganizationID)
    if err != nil {
        return "", errors.New("organization_not_found")
    }

    // 4. Gerar novo access token
    permissions := s.getPermissionsForRole(member.Role)

    accessToken, err := s.jwtService.GenerateAccessToken(
        user.ID,
        user.Email.Value(),
        member.OrganizationID,
        member.Organization.Name,
        string(member.Role),
        permissions,
    )

    return accessToken, err
}

// Logout invalida refresh token
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
    claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
    if err != nil {
        return err
    }

    return s.tokenStore.Revoke(ctx, claims.ID)
}

// getPermissionsForRole retorna permissões para uma role
func (s *AuthService) getPermissionsForRole(role entities.Role) []string {
    switch role {
    case entities.RoleAdmin:
        return []string{"*:*"} // Wildcard - todas as permissões
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

// extractJTI extrai o JTI (ID do token) de um JWT
func extractJTI(tokenString string) string {
    // Parse token sem validar (apenas para extrair claims)
    token, _ := jwt.Parse(tokenString, nil)
    if claims, ok := token.Claims.(jwt.MapClaims); ok {
        if jti, ok := claims["jti"].(string); ok {
            return jti
        }
    }
    return ""
}
```

---

### 4.5 Activation Service (Fluxo Simplificado)

**Novos Endpoints para Fluxo Simplificado**:
- POST /auth/register-complete
- GET /activate
- POST /auth/resend-activation

```go
// internal/services/activation_service.go
package services

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "time"

    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/valueobjects"
    "avantpro-backend/internal/infrastructure/auth"
)

type ActivationService struct {
    userRepo         entities.UserRepository
    orgRepo          entities.OrganizationRepository
    orgMemberRepo    entities.OrganizationMemberRepository
    activationRepo   ActivationTokenRepository
    emailGateway     EmailGateway
    jwtService       *auth.JWTService
    uow              UnitOfWork
}

// RegisterComplete cria User + Organization + OrganizationMember atomicamente
func (s *ActivationService) RegisterComplete(
    ctx context.Context,
    email, password, organizationName string,
) error {
    // Iniciar transaction
    return s.uow.Execute(ctx, func(ctx context.Context) error {
        // 1. Criar User (inactive)
        user, err := entities.NewUser(email, password)
        if err != nil {
            return err
        }
        user.Status = "inactive"

        if err := s.userRepo.Create(ctx, user); err != nil {
            return err
        }

        // 2. Criar Organization (active)
        org := entities.NewOrganization(organizationName)
        org.Status = "active"

        if err := s.orgRepo.Create(ctx, org); err != nil {
            return err
        }

        // 3. Criar OrganizationMember (owner)
        member := entities.NewOrganizationMember(user.ID, org.ID, "owner")

        if err := s.orgMemberRepo.Create(ctx, member); err != nil {
            return err
        }

        // 4. Gerar token de ativação (24h)
        token := generateSecureToken()
        activationToken := &ActivationToken{
            UserID:    user.ID,
            Token:     token,
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
            CreatedAt: time.Now().Unix(),
        }

        if err := s.activationRepo.Create(ctx, activationToken); err != nil {
            return err
        }

        // 5. Enviar email de ativação
        activationURL := fmt.Sprintf("https://app.avantpro.com.br/activate?token=%s", token)
        emailData := map[string]string{
            "email":             user.Email,
            "organization_name": org.Name,
            "activation_url":    activationURL,
        }

        return s.emailGateway.SendActivationEmail(ctx, emailData)
    })
}

// Activate ativa conta + login automático
func (s *ActivationService) Activate(
    ctx context.Context,
    token string,
) (*ActivationResponse, error) {
    // 1. Validar token
    activationToken, err := s.activationRepo.FindByToken(ctx, token)
    if err != nil {
        return nil, errors.New("invalid_token")
    }

    if activationToken.UsedAt != nil {
        return nil, errors.New("token_already_used")
    }

    if time.Now().Unix() > activationToken.ExpiresAt {
        return nil, errors.New("token_expired")
    }

    // 2. Buscar user
    user, err := s.userRepo.FindByID(ctx, activationToken.UserID)
    if err != nil {
        return nil, err
    }

    if user.Status == "active" {
        return nil, errors.New("account_already_active")
    }

    // 3. Buscar organization do user
    orgMember, err := s.orgMemberRepo.FindByUserID(ctx, user.ID)
    if err != nil {
        return nil, err
    }

    org, err := s.orgRepo.FindByID(ctx, orgMember.OrganizationID)
    if err != nil {
        return nil, err
    }

    // 4. Ativar conta atomicamente
    return s.uow.Execute(ctx, func(ctx context.Context) (*ActivationResponse, error) {
        // Ativar user
        user.Status = "active"
        now := time.Now()
        user.EmailVerifiedAt = &now

        if err := s.userRepo.Update(ctx, user); err != nil {
            return nil, err
        }

        // Marcar token como usado
        usedAt := time.Now().Unix()
        activationToken.UsedAt = &usedAt

        if err := s.activationRepo.Update(ctx, activationToken); err != nil {
            return nil, err
        }

        // 5. Gerar JWT final (com organization_id)
        accessToken, err := s.jwtService.GenerateAccessToken(
            user.ID,
            user.Email,
            org.ID,
            org.Name,
            orgMember.Role,
            getPermissionsForRole(orgMember.Role),
        )
        if err != nil {
            return nil, err
        }

        refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID, org.ID)
        if err != nil {
            return nil, err
        }

        // 6. Retornar resposta com login automático
        return &ActivationResponse{
            AccessToken:  accessToken,
            RefreshToken: refreshToken,
            User: UserResponse{
                ID:              user.ID,
                Email:           user.Email,
                EmailVerifiedAt: user.EmailVerifiedAt.Unix(),
            },
            Organization: OrganizationResponse{
                ID:   org.ID,
                Name: org.Name,
                Role: orgMember.Role,
            },
            RedirectTo: "/dashboard?welcome=true",
        }, nil
    })
}

// ResendActivation reenv ia email de ativação
func (s *ActivationService) ResendActivation(
    ctx context.Context,
    email string,
) error {
    // 1. Buscar user
    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil {
        // Por segurança, retorna sucesso mesmo se email não existe
        return nil
    }

    // 2. Verificar se já está ativo
    if user.Status == "active" {
        return errors.New("account_already_active")
    }

    // 3. Buscar organization
    orgMember, err := s.orgMemberRepo.FindByUserID(ctx, user.ID)
    if err != nil {
        return err
    }

    org, err := s.orgRepo.FindByID(ctx, orgMember.OrganizationID)
    if err != nil {
        return err
    }

    // 4. Invalidar tokens anteriores + criar novo
    return s.uow.Execute(ctx, func(ctx context.Context) error {
        // Marcar tokens antigos como usados (apenas mais recente é válido)
        if err := s.activationRepo.InvalidateByUserID(ctx, user.ID); err != nil {
            return err
        }

        // Gerar novo token
        token := generateSecureToken()
        activationToken := &ActivationToken{
            UserID:    user.ID,
            Token:     token,
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
            CreatedAt: time.Now().Unix(),
        }

        if err := s.activationRepo.Create(ctx, activationToken); err != nil {
            return err
        }

        // Enviar email
        activationURL := fmt.Sprintf("https://app.avantpro.com.br/activate?token=%s", token)
        emailData := map[string]string{
            "email":             user.Email,
            "organization_name": org.Name,
            "activation_url":    activationURL,
        }

        return s.emailGateway.SendActivationEmail(ctx, emailData)
    })
}

// Helpers
func generateSecureToken() string {
    bytes := make([]byte, 32)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}

func getPermissionsForRole(role string) []string {
    // Placeholder - implementação real virá de RBAC spec
    if role == "owner" {
        return []string{"*:*"}
    }
    return []string{}
}

// DTOs
type ActivationResponse struct {
    AccessToken  string               `json:"access_token"`
    RefreshToken string               `json:"refresh_token"`
    User         UserResponse         `json:"user"`
    Organization OrganizationResponse `json:"organization"`
    RedirectTo   string               `json:"redirect_to"`
}

type UserResponse struct {
    ID              string `json:"id"`
    Email           string `json:"email"`
    EmailVerifiedAt int64  `json:"email_verified_at"`
}

type OrganizationResponse struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Role string `json:"role"`
}

// Repository Interface
type ActivationTokenRepository interface {
    Create(ctx context.Context, token *ActivationToken) error
    Update(ctx context.Context, token *ActivationToken) error
    FindByToken(ctx context.Context, token string) (*ActivationToken, error)
    FindByUserID(ctx context.Context, userID string) (*ActivationToken, error)
    InvalidateByUserID(ctx context.Context, userID string) error
}

type ActivationToken struct {
    ID        string
    UserID    string
    Token     string
    ExpiresAt int64
    UsedAt    *int64
    CreatedAt int64
}
```

**HTTP Handler**:

```go
// internal/handlers/http/activation_handler.go
package http

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/services"
)

type ActivationHandler struct {
    activationService *services.ActivationService
}

func NewActivationHandler(svc *services.ActivationService) *ActivationHandler {
    return &ActivationHandler{activationService: svc}
}

// POST /auth/register-complete
func (h *ActivationHandler) RegisterComplete(c *gin.Context) {
    var req struct {
        Email            string `json:"email" binding:"required,email"`
        Password         string `json:"password" binding:"required,min=8"`
        OrganizationName string `json:"organization_name" binding:"required,min=2,max=100"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "details": err.Error()})
        return
    }

    err := h.activationService.RegisterComplete(c.Request.Context(), req.Email, req.Password, req.OrganizationName)
    if err != nil {
        c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message":           "Enviamos um email de ativação. Verifique sua caixa de entrada.",
        "email":             req.Email,
        "organization_name": req.OrganizationName,
    })
}

// GET /activate?token=xyz
func (h *ActivationHandler) Activate(c *gin.Context) {
    token := c.Query("token")
    if token == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "token_required"})
        return
    }

    response, err := h.activationService.Activate(c.Request.Context(), token)
    if err != nil {
        switch err.Error() {
        case "invalid_token":
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_token"})
        case "token_expired":
            c.JSON(http.StatusGone, gin.H{"error": "token_expired"})
        case "account_already_active":
            c.JSON(http.StatusConflict, gin.H{"error": "account_already_active", "message": "Sua conta já está ativa. Faça login."})
        default:
            c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
        }
        return
    }

    c.JSON(http.StatusOK, response)
}

// POST /auth/resend-activation
func (h *ActivationHandler) ResendActivation(c *gin.Context) {
    var req struct {
        Email string `json:"email" binding:"required,email"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed"})
        return
    }

    err := h.activationService.ResendActivation(c.Request.Context(), req.Email)
    if err != nil {
        if err.Error() == "account_already_active" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "account_already_active"})
            return
        }
        // Por segurança, sempre retorna sucesso
    }

    c.JSON(http.StatusOK, gin.H{"message": "Novo email de ativação enviado"})
}
```

**Rotas**:

```go
// cmd/api/main.go (ou internal/handlers/http/router.go)
authGroup := router.Group("/auth")
{
    authGroup.POST("/register-complete", activationHandler.RegisterComplete)
    authGroup.POST("/resend-activation", activationHandler.ResendActivation)
}

router.GET("/activate", activationHandler.Activate)
```

---

## 5. Middleware de Autenticação

### 5.1 Middleware HTTP

```go
// internal/handlers/http/middleware/auth_middleware.go
package middleware

import (
    "context"
    "strings"
    "net/http"

    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/infrastructure/auth"
)

type contextKey string

const (
    UserIDKey           contextKey = "user_id"
    EmailKey            contextKey = "email"
    OrganizationIDKey   contextKey = "organization_id"
    RoleKey             contextKey = "role"
    PermissionsKey      contextKey = "permissions"
)

// AuthMiddleware valida JWT e extrai claims
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Extrair token do header Authorization
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "missing_authorization_header",
            })
            c.Abort()
            return
        }

        // 2. Validar formato "Bearer <token>"
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "invalid_authorization_format",
            })
            c.Abort()
            return
        }

        tokenString := parts[1]

        // 3. Validar e parsear JWT
        claims, err := jwtService.ValidateAccessToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "invalid_token",
            })
            c.Abort()
            return
        }

        // 4. Adicionar claims ao contexto
        ctx := context.WithValue(c.Request.Context(), UserIDKey, claims.UserID)
        ctx = context.WithValue(ctx, EmailKey, claims.Email)
        ctx = context.WithValue(ctx, OrganizationIDKey, claims.OrganizationID)
        ctx = context.WithValue(ctx, RoleKey, claims.Role)
        ctx = context.WithValue(ctx, PermissionsKey, claims.Permissions)

        c.Request = c.Request.WithContext(ctx)

        c.Next()
    }
}

// RequirePermission middleware que valida permissão específica
func RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        permissions, ok := c.Request.Context().Value(PermissionsKey).([]string)
        if !ok {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "forbidden",
            })
            c.Abort()
            return
        }

        // Verificar wildcard (admin)
        for _, p := range permissions {
            if p == "*:*" {
                c.Next()
                return
            }
        }

        // Verificar permissão específica
        hasPermission := false
        for _, p := range permissions {
            if p == permission {
                hasPermission = true
                break
            }
        }

        if !hasPermission {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "forbidden",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

---

## 6. OAuth2 Integration

### 6.1 Google OAuth2

```go
// internal/infrastructure/auth/oauth2_google.go
package auth

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

type GoogleOAuth2Config struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
}

type GoogleUserInfo struct {
    ID            string `json:"id"`
    Email         string `json:"email"`
    VerifiedEmail bool   `json:"verified_email"`
    Name          string `json:"name"`
    GivenName     string `json:"given_name"`
    FamilyName    string `json:"family_name"`
    Picture       string `json:"picture"`
}

type GoogleOAuth2Service struct {
    config *oauth2.Config
}

func NewGoogleOAuth2Service(cfg GoogleOAuth2Config) *GoogleOAuth2Service {
    return &GoogleOAuth2Service{
        config: &oauth2.Config{
            ClientID:     cfg.ClientID,
            ClientSecret: cfg.ClientSecret,
            RedirectURL:  cfg.RedirectURL,
            Scopes: []string{
                "https://www.googleapis.com/auth/userinfo.email",
                "https://www.googleapis.com/auth/userinfo.profile",
            },
            Endpoint: google.Endpoint,
        },
    }
}

// GetAuthURL retorna URL de autorização do Google
func (s *GoogleOAuth2Service) GetAuthURL(state string) string {
    return s.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode troca código por tokens
func (s *GoogleOAuth2Service) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
    return s.config.Exchange(ctx, code)
}

// GetUserInfo obtém informações do usuário do Google
func (s *GoogleOAuth2Service) GetUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
    client := s.config.Client(ctx, token)

    resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to get user info: %s", resp.Status)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var userInfo GoogleUserInfo
    if err := json.Unmarshal(body, &userInfo); err != nil {
        return nil, err
    }

    return &userInfo, nil
}
```

### 6.2 OAuth2 Handler

```go
// internal/handlers/http/oauth_handler.go
package httphandlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/services"
    "avantpro-backend/internal/infrastructure/auth"
)

type OAuth2Handler struct {
    authService   *services.AuthService
    googleService *auth.GoogleOAuth2Service
}

func NewOAuth2Handler(
    authService *services.AuthService,
    googleService *auth.GoogleOAuth2Service,
) *OAuth2Handler {
    return &OAuth2Handler{
        authService:   authService,
        googleService: googleService,
    }
}

// GoogleLogin inicia fluxo OAuth Google
func (h *OAuth2Handler) GoogleLogin(c *gin.Context) {
    state := generateState() // CSRF protection

    // Armazenar state em session/cookie para validar no callback
    c.SetCookie("oauth_state", state, 600, "/", "", true, true)

    url := h.googleService.GetAuthURL(state)
    c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback processa callback do Google
func (h *OAuth2Handler) GoogleCallback(c *gin.Context) {
    // 1. Validar state (CSRF protection)
    state := c.Query("state")
    cookieState, _ := c.Cookie("oauth_state")

    if state != cookieState {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "invalid_state",
        })
        return
    }

    // 2. Trocar code por token
    code := c.Query("code")
    token, err := h.googleService.ExchangeCode(c.Request.Context(), code)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "failed_to_exchange_code",
        })
        return
    }

    // 3. Obter informações do usuário
    userInfo, err := h.googleService.GetUserInfo(c.Request.Context(), token)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "failed_to_get_user_info",
        })
        return
    }

    // 4. Criar/atualizar usuário no banco + gerar JWT próprio
    response, err := h.authService.LoginOrRegisterWithOAuth(
        c.Request.Context(),
        userInfo.Email,
        userInfo.Name,
        userInfo.Picture,
    )

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, response)
}

func generateState() string {
    return uuid.New().String()
}
```

---

## 7. Testes

### 7.1 Testes de JWT

```go
// internal/infrastructure/auth/jwt_test.go
package auth_test

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "avantpro-backend/internal/infrastructure/auth"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
    jwtService := auth.NewJWTService("secret", "avantpro", "avantpro-api")

    token, err := jwtService.GenerateAccessToken(
        "user-123",
        "user@example.com",
        "org-abc",
        "Empresa ABC",
        "admin",
        []string{"*:*"},
    )

    assert.NoError(t, err)
    assert.NotEmpty(t, token)

    // Validar token
    claims, err := jwtService.ValidateAccessToken(token)
    assert.NoError(t, err)
    assert.Equal(t, "user-123", claims.UserID)
    assert.Equal(t, "user@example.com", claims.Email)
    assert.Equal(t, "org-abc", claims.OrganizationID)
    assert.Equal(t, "admin", claims.Role)
}

func TestValidateExpiredToken(t *testing.T) {
    // TODO: implementar teste com token expirado
}

func TestValidateInvalidSignature(t *testing.T) {
    jwtService := auth.NewJWTService("secret", "avantpro", "avantpro-api")
    wrongService := auth.NewJWTService("wrong-secret", "avantpro", "avantpro-api")

    token, _ := jwtService.GenerateAccessToken("user-123", "user@example.com", "org-abc", "Org", "admin", []string{})

    // Tentar validar com chave errada
    _, err := wrongService.ValidateAccessToken(token)
    assert.Error(t, err)
}
```

---

## 8. Configuração

### 8.1 Variáveis de Ambiente

```bash
# .env
JWT_SECRET=your-super-secret-key-change-in-production
JWT_ISSUER=avantpro
JWT_AUDIENCE=avantpro-api

GOOGLE_OAUTH_CLIENT_ID=your-google-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-google-client-secret
GOOGLE_OAUTH_REDIRECT_URL=http://localhost:8080/auth/oauth/google/callback

REDIS_URL=redis://localhost:6379
```

### 8.2 Inicialização

```go
// cmd/api/main.go
func main() {
    cfg := config.Load()

    // JWT Service
    jwtService := auth.NewJWTService(
        cfg.JWTSecret,
        cfg.JWTIssuer,
        cfg.JWTAudience,
    )

    // Redis client
    redisClient := redis.NewClient(&redis.Options{
        Addr: cfg.RedisURL,
    })

    // Refresh token store
    tokenStore := auth.NewRefreshTokenStore(redisClient)

    // Google OAuth2
    googleService := auth.NewGoogleOAuth2Service(auth.GoogleOAuth2Config{
        ClientID:     cfg.GoogleOAuthClientID,
        ClientSecret: cfg.GoogleOAuthClientSecret,
        RedirectURL:  cfg.GoogleOAuthRedirectURL,
    })

    // Services
    authService := services.NewAuthService(userRepo, orgMemberRepo, jwtService, tokenStore)

    // Handlers
    authHandler := httphandlers.NewAuthHandler(authService)
    oauthHandler := httphandlers.NewOAuth2Handler(authService, googleService)

    // Routes
    router := gin.Default()

    // Public routes
    router.POST("/auth/login", authHandler.Login)
    router.POST("/auth/register", authHandler.Register)
    router.POST("/auth/refresh", authHandler.Refresh)
    router.GET("/auth/oauth/google", oauthHandler.GoogleLogin)
    router.GET("/auth/oauth/google/callback", oauthHandler.GoogleCallback)

    // Protected routes
    protected := router.Group("/")
    protected.Use(middleware.AuthMiddleware(jwtService))
    {
        protected.POST("/auth/logout", authHandler.Logout)
        protected.GET("/auth/me", authHandler.Me)
    }

    router.Run(":8080")
}
```

---

## 9. Segurança

### 9.1 Checklist de Segurança

- ✅ JWT assinado com HMAC-SHA256
- ✅ Secret key forte (256+ bits) em variável de ambiente
- ✅ Access token curto (15 min) - limita janela de ataque
- ✅ Refresh token longo mas revogável (Redis)
- ✅ JTI único por token (previne replay attacks)
- ✅ Validação de issuer e audience
- ✅ CSRF protection em OAuth (state parameter)
- ✅ httpOnly cookies para refresh tokens
- ✅ HTTPS obrigatório em produção

### 9.2 Rotação de Secrets

```go
// Implementar rotação de chaves JWT
// TODO: suportar múltiplas chaves para rotação sem downtime
```

---

## 10. Referências

**Specs Relacionadas**:
- `specs/functional/auth.md` - Requisitos funcionais
- `specs/technical/security.md` - Especificação de segurança

**Bibliotecas**:
- `github.com/golang-jwt/jwt/v5` - JWT
- `golang.org/x/oauth2` - OAuth2
- `github.com/redis/go-redis/v9` - Redis client
