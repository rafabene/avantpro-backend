# Autenticação e Autorização

**Versão**: 2.0
**Data**: 05/11/2025

---

## 1. Visão Geral

Sistema de autenticação e autorização multi-camadas:
- **JWT** para autenticação stateless (Access + Refresh tokens)
- **OAuth2/OIDC** para login social (Google, GitHub, etc)
- **RBAC** para controle de acesso baseado em roles/permissions

---

## 2. Arquitetura de Autenticação

```
┌──────────────────────────────────────────────────────────┐
│                        CLIENT                             │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────────────┐
│               AUTH ENDPOINTS                              │
│                                                           │
│  POST /auth/login         - Login com email/password     │
│  POST /auth/register      - Registro de usuário          │
│  POST /auth/refresh       - Renovar access token         │
│  POST /auth/logout        - Logout (invalidar tokens)    │
│  GET  /auth/oauth/google  - Iniciar OAuth Google         │
│  GET  /auth/oauth/callback- Callback OAuth               │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────────────┐
│                  AUTH SERVICE                             │
│                                                           │
│  • Valida credenciais                                    │
│  • Gera JWT tokens                                       │
│  • Gerencia refresh tokens                               │
│  • Integra com OAuth providers                           │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────────────┐
│              TOKEN MANAGEMENT                             │
│                                                           │
│  Access Token:  Short-lived (15min)                      │
│  Refresh Token: Long-lived (7 days)                      │
│  Storage:       Redis (refresh tokens)                   │
└──────────────────────────────────────────────────────────┘
```

---

## 3. JWT (JSON Web Tokens)

### 3.1 Estrutura de Tokens

#### Access Token (15 minutos)
```json
{
  "sub": "user_id_123",
  "email": "user@example.com",
  "role": "admin",
  "permissions": ["users.read", "users.write"],
  "iat": 1699123456,
  "exp": 1699124356
}
```

#### Refresh Token (7 dias)
```json
{
  "sub": "user_id_123",
  "jti": "token_id_xyz",
  "iat": 1699123456,
  "exp": 1699728256
}
```

### 3.2 Implementação JWT

```go
// internal/infrastructure/auth/jwt.go
package auth

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
    "avantpro-backend/internal/domain/entities"
)

type TokenType string

const (
    TokenTypeAccess  TokenType = "access"
    TokenTypeRefresh TokenType = "refresh"
)

type JWTManager struct {
    secretKey     string
    accessExpiry  time.Duration
    refreshExpiry time.Duration
}

func NewJWTManager(secretKey string) *JWTManager {
    return &JWTManager{
        secretKey:     secretKey,
        accessExpiry:  15 * time.Minute,
        refreshExpiry: 7 * 24 * time.Hour,
    }
}

type AccessTokenClaims struct {
    UserID      string   `json:"sub"`
    Email       string   `json:"email"`
    Role        string   `json:"role"`
    Permissions []string `json:"permissions"`
    jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
    UserID string `json:"sub"`
    JTI    string `json:"jti"` // Token ID para revogação
    jwt.RegisteredClaims
}

// GenerateAccessToken gera token de acesso
func (m *JWTManager) GenerateAccessToken(user *entities.User) (string, error) {
    claims := AccessTokenClaims{
        UserID:      user.ID,
        Email:       user.Email.String(),
        Role:        string(user.Role),
        Permissions: user.GetPermissions(),
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(m.secretKey))
}

// GenerateRefreshToken gera token de refresh
func (m *JWTManager) GenerateRefreshToken(userID string) (string, string, error) {
    jti := generateUUID() // Token ID único

    claims := RefreshTokenClaims{
        UserID: userID,
        JTI:    jti,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(m.secretKey))
    return tokenString, jti, err
}

// ValidateAccessToken valida e extrai claims do access token
func (m *JWTManager) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(m.secretKey), nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*AccessTokenClaims); ok && token.Valid {
        return claims, nil
    }

    return nil, jwt.ErrSignatureInvalid
}

// ValidateRefreshToken valida refresh token
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(m.secretKey), nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*RefreshTokenClaims); ok && token.Valid {
        return claims, nil
    }

    return nil, jwt.ErrSignatureInvalid
}
```

### 3.3 Refresh Token Storage (Redis)

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
    redis *redis.Client
}

func NewRefreshTokenStore(redis *redis.Client) *RefreshTokenStore {
    return &RefreshTokenStore{redis: redis}
}

// StoreRefreshToken armazena refresh token no Redis
func (s *RefreshTokenStore) StoreRefreshToken(ctx context.Context, userID, jti string, expiry time.Duration) error {
    key := fmt.Sprintf("refresh_token:%s:%s", userID, jti)
    return s.redis.Set(ctx, key, "valid", expiry).Err()
}

// IsRefreshTokenValid verifica se refresh token está válido
func (s *RefreshTokenStore) IsRefreshTokenValid(ctx context.Context, userID, jti string) (bool, error) {
    key := fmt.Sprintf("refresh_token:%s:%s", userID, jti)
    result, err := s.redis.Get(ctx, key).Result()

    if err == redis.Nil {
        return false, nil // Token não existe
    }
    if err != nil {
        return false, err
    }

    return result == "valid", nil
}

// RevokeRefreshToken revoga um refresh token
func (s *RefreshTokenStore) RevokeRefreshToken(ctx context.Context, userID, jti string) error {
    key := fmt.Sprintf("refresh_token:%s:%s", userID, jti)
    return s.redis.Del(ctx, key).Err()
}

// RevokeAllUserTokens revoga todos os tokens de um usuário
func (s *RefreshTokenStore) RevokeAllUserTokens(ctx context.Context, userID string) error {
    pattern := fmt.Sprintf("refresh_token:%s:*", userID)

    iter := s.redis.Scan(ctx, 0, pattern, 0).Iterator()
    for iter.Next(ctx) {
        if err := s.redis.Del(ctx, iter.Val()).Err(); err != nil {
            return err
        }
    }

    return iter.Err()
}
```

---

## 4. Auth Service

```go
// internal/services/auth_service.go
package services

import (
    "context"
    "errors"
    "golang.org/x/crypto/bcrypt"
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/repositories"
    "avantpro-backend/internal/infrastructure/auth"
)

type AuthService struct {
    userRepo         repositories.UserRepository
    jwtManager       *auth.JWTManager
    refreshTokenStore *auth.RefreshTokenStore
}

func NewAuthService(
    userRepo repositories.UserRepository,
    jwtManager *auth.JWTManager,
    refreshTokenStore *auth.RefreshTokenStore,
) *AuthService {
    return &AuthService{
        userRepo:          userRepo,
        jwtManager:        jwtManager,
        refreshTokenStore: refreshTokenStore,
    }
}

type LoginResult struct {
    AccessToken  string
    RefreshToken string
    User         *entities.User
}

// Login autentica usuário e retorna tokens
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
    // Buscar usuário
    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return nil, err
    }
    if user == nil {
        return nil, errors.New("invalid credentials")
    }

    // Verificar senha
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return nil, errors.New("invalid credentials")
    }

    // Gerar access token
    accessToken, err := s.jwtManager.GenerateAccessToken(user)
    if err != nil {
        return nil, err
    }

    // Gerar refresh token
    refreshToken, jti, err := s.jwtManager.GenerateRefreshToken(user.ID)
    if err != nil {
        return nil, err
    }

    // Armazenar refresh token
    if err := s.refreshTokenStore.StoreRefreshToken(ctx, user.ID, jti, 7*24*time.Hour); err != nil {
        return nil, err
    }

    return &LoginResult{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        User:         user,
    }, nil
}

// RefreshToken renova access token usando refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
    // Validar refresh token
    claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
    if err != nil {
        return "", errors.New("invalid refresh token")
    }

    // Verificar se token não foi revogado
    valid, err := s.refreshTokenStore.IsRefreshTokenValid(ctx, claims.UserID, claims.JTI)
    if err != nil {
        return "", err
    }
    if !valid {
        return "", errors.New("refresh token revoked")
    }

    // Buscar usuário
    user, err := s.userRepo.FindByID(ctx, claims.UserID)
    if err != nil {
        return "", err
    }
    if user == nil {
        return "", errors.New("user not found")
    }

    // Gerar novo access token
    return s.jwtManager.GenerateAccessToken(user)
}

// Logout revoga refresh token
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
    claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
    if err != nil {
        return err
    }

    return s.refreshTokenStore.RevokeRefreshToken(ctx, claims.UserID, claims.JTI)
}

// LogoutAll revoga todos os tokens do usuário
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
    return s.refreshTokenStore.RevokeAllUserTokens(ctx, userID)
}
```

---

## 5. Middlewares

### 5.1 Authentication Middleware

```go
// internal/handlers/middleware/auth.go
package middleware

import (
    "net/http"
    "strings"
    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/infrastructure/auth"
)

type AuthMiddleware struct {
    jwtManager *auth.JWTManager
}

func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
    return &AuthMiddleware{jwtManager: jwtManager}
}

// Authenticate verifica JWT token
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extrair token do header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }

        // Bearer token
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
            c.Abort()
            return
        }

        tokenString := parts[1]

        // Validar token
        claims, err := m.jwtManager.ValidateAccessToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        // Adicionar claims ao contexto
        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)
        c.Set("user_role", claims.Role)
        c.Set("user_permissions", claims.Permissions)

        c.Next()
    }
}

// Optional permite requests com ou sem autenticação
func (m *AuthMiddleware) Optional() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.Next()
            return
        }

        parts := strings.Split(authHeader, " ")
        if len(parts) == 2 && parts[0] == "Bearer" {
            claims, err := m.jwtManager.ValidateAccessToken(parts[1])
            if err == nil {
                c.Set("user_id", claims.UserID)
                c.Set("user_email", claims.Email)
                c.Set("user_role", claims.Role)
                c.Set("user_permissions", claims.Permissions)
            }
        }

        c.Next()
    }
}
```

### 5.2 RBAC Middleware

```go
// internal/handlers/middleware/rbac.go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

type RBACMiddleware struct{}

func NewRBACMiddleware() *RBACMiddleware {
    return &RBACMiddleware{}
}

// RequireRole verifica se usuário tem role específica
func (m *RBACMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole, exists := c.Get("user_role")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }

        role := userRole.(string)
        for _, requiredRole := range roles {
            if role == requiredRole {
                c.Next()
                return
            }
        }

        c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
        c.Abort()
    }
}

// RequirePermission verifica se usuário tem permissão específica
func (m *RBACMiddleware) RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userPermissions, exists := c.Get("user_permissions")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }

        permissions := userPermissions.([]string)
        for _, p := range permissions {
            if p == permission {
                c.Next()
                return
            }
        }

        c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
        c.Abort()
    }
}

// RequireAnyPermission verifica se usuário tem pelo menos uma das permissões
func (m *RBACMiddleware) RequireAnyPermission(permissions ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userPermissions, exists := c.Get("user_permissions")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }

        userPerms := userPermissions.([]string)
        for _, required := range permissions {
            for _, userPerm := range userPerms {
                if userPerm == required {
                    c.Next()
                    return
                }
            }
        }

        c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
        c.Abort()
    }
}
```

---

## 6. OAuth2/OIDC

### 6.1 OAuth2 Provider Configuration

```go
// internal/infrastructure/auth/oauth2.go
package auth

import (
    "context"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "golang.org/x/oauth2/github"
)

type OAuth2Provider string

const (
    ProviderGoogle OAuth2Provider = "google"
    ProviderGitHub OAuth2Provider = "github"
)

type OAuth2Manager struct {
    configs map[OAuth2Provider]*oauth2.Config
}

func NewOAuth2Manager(
    googleClientID, googleClientSecret string,
    githubClientID, githubClientSecret string,
    redirectURL string,
) *OAuth2Manager {
    return &OAuth2Manager{
        configs: map[OAuth2Provider]*oauth2.Config{
            ProviderGoogle: {
                ClientID:     googleClientID,
                ClientSecret: googleClientSecret,
                RedirectURL:  redirectURL + "/auth/oauth/google/callback",
                Scopes:       []string{"email", "profile"},
                Endpoint:     google.Endpoint,
            },
            ProviderGitHub: {
                ClientID:     githubClientID,
                ClientSecret: githubClientSecret,
                RedirectURL:  redirectURL + "/auth/oauth/github/callback",
                Scopes:       []string{"user:email"},
                Endpoint:     github.Endpoint,
            },
        },
    }
}

// GetAuthURL retorna URL para iniciar OAuth flow
func (m *OAuth2Manager) GetAuthURL(provider OAuth2Provider, state string) (string, error) {
    config, ok := m.configs[provider]
    if !ok {
        return "", errors.New("unknown provider")
    }

    return config.AuthCodeURL(state), nil
}

// ExchangeCode troca authorization code por access token
func (m *OAuth2Manager) ExchangeCode(ctx context.Context, provider OAuth2Provider, code string) (*oauth2.Token, error) {
    config, ok := m.configs[provider]
    if !ok {
        return nil, errors.New("unknown provider")
    }

    return config.Exchange(ctx, code)
}
```

### 6.2 OAuth2 User Info

```go
// internal/infrastructure/auth/oauth2_userinfo.go
package auth

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "golang.org/x/oauth2"
)

type OAuthUserInfo struct {
    ID       string
    Email    string
    Name     string
    Picture  string
    Provider string
}

// GetGoogleUserInfo busca informações do usuário do Google
func GetGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
    client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
    resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var googleUser struct {
        ID      string `json:"id"`
        Email   string `json:"email"`
        Name    string `json:"name"`
        Picture string `json:"picture"`
    }

    if err := json.Unmarshal(data, &googleUser); err != nil {
        return nil, err
    }

    return &OAuthUserInfo{
        ID:       googleUser.ID,
        Email:    googleUser.Email,
        Name:     googleUser.Name,
        Picture:  googleUser.Picture,
        Provider: "google",
    }, nil
}

// GetGitHubUserInfo busca informações do usuário do GitHub
func GetGitHubUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
    client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
    resp, err := client.Get("https://api.github.com/user")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var githubUser struct {
        ID        int64  `json:"id"`
        Login     string `json:"login"`
        Name      string `json:"name"`
        Email     string `json:"email"`
        AvatarURL string `json:"avatar_url"`
    }

    if err := json.Unmarshal(data, &githubUser); err != nil {
        return nil, err
    }

    return &OAuthUserInfo{
        ID:       fmt.Sprintf("%d", githubUser.ID),
        Email:    githubUser.Email,
        Name:     githubUser.Name,
        Picture:  githubUser.AvatarURL,
        Provider: "github",
    }, nil
}
```

### 6.3 OAuth Handler

```go
// internal/handlers/http/auth_handler.go
package http

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/services"
    "avantpro-backend/internal/infrastructure/auth"
)

type AuthHandler struct {
    authService   *services.AuthService
    oauth2Manager *auth.OAuth2Manager
}

// LoginWithGoogle inicia OAuth flow com Google
func (h *AuthHandler) LoginWithGoogle(c *gin.Context) {
    state := generateRandomState() // Armazenar em session/cookie para validar depois

    url, err := h.oauth2Manager.GetAuthURL(auth.ProviderGoogle, state)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate auth url"})
        return
    }

    c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback processa callback do Google OAuth
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
    // Validar state
    state := c.Query("state")
    // TODO: validar state contra session/cookie

    code := c.Query("code")
    if code == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
        return
    }

    // Trocar code por token
    token, err := h.oauth2Manager.ExchangeCode(c.Request.Context(), auth.ProviderGoogle, code)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange code"})
        return
    }

    // Buscar informações do usuário
    userInfo, err := auth.GetGoogleUserInfo(c.Request.Context(), token)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
        return
    }

    // Criar ou buscar usuário no banco
    result, err := h.authService.LoginOrRegisterOAuth(c.Request.Context(), userInfo)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
        return
    }

    // Retornar tokens
    c.JSON(http.StatusOK, gin.H{
        "access_token":  result.AccessToken,
        "refresh_token": result.RefreshToken,
        "user":          result.User,
    })
}
```

---

## 7. RBAC (Role-Based Access Control)

### 7.1 Domain Model

```go
// internal/domain/entities/role.go
package entities

type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
    RoleGuest Role = "guest"
)

type Permission string

const (
    // User permissions
    PermissionUserRead   Permission = "users.read"
    PermissionUserWrite  Permission = "users.write"
    PermissionUserDelete Permission = "users.delete"

    // Subscription permissions
    PermissionSubscriptionRead  Permission = "subscriptions.read"
    PermissionSubscriptionWrite Permission = "subscriptions.write"

    // Payment permissions
    PermissionPaymentRead  Permission = "payments.read"
    PermissionPaymentWrite Permission = "payments.write"
)

// RolePermissions mapeia roles para suas permissões
var RolePermissions = map[Role][]Permission{
    RoleAdmin: {
        PermissionUserRead,
        PermissionUserWrite,
        PermissionUserDelete,
        PermissionSubscriptionRead,
        PermissionSubscriptionWrite,
        PermissionPaymentRead,
        PermissionPaymentWrite,
    },
    RoleUser: {
        PermissionUserRead,
        PermissionSubscriptionRead,
        PermissionSubscriptionWrite,
    },
    RoleGuest: {
        PermissionUserRead,
    },
}

// GetPermissions retorna permissões de um role
func (r Role) GetPermissions() []Permission {
    return RolePermissions[r]
}

// HasPermission verifica se role tem permissão
func (r Role) HasPermission(permission Permission) bool {
    permissions := RolePermissions[r]
    for _, p := range permissions {
        if p == permission {
            return true
        }
    }
    return false
}
```

```go
// internal/domain/entities/user.go
package entities

func (u *User) GetPermissions() []string {
    perms := u.Role.GetPermissions()
    result := make([]string, len(perms))
    for i, p := range perms {
        result[i] = string(p)
    }
    return result
}

func (u *User) HasPermission(permission Permission) bool {
    return u.Role.HasPermission(permission)
}
```

### 7.2 Uso nas Routes

```go
// cmd/api/routes.go
package main

import (
    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/handlers/http"
    "avantpro-backend/internal/handlers/middleware"
)

func setupRoutes(
    router *gin.Engine,
    authMiddleware *middleware.AuthMiddleware,
    rbacMiddleware *middleware.RBACMiddleware,
    userHandler *http.UserHandler,
) {
    // Public routes
    public := router.Group("/api/v1")
    {
        public.POST("/auth/login", authHandler.Login)
        public.POST("/auth/register", authHandler.Register)
        public.GET("/auth/oauth/google", authHandler.LoginWithGoogle)
        public.GET("/auth/oauth/google/callback", authHandler.GoogleCallback)
    }

    // Authenticated routes
    authenticated := router.Group("/api/v1")
    authenticated.Use(authMiddleware.Authenticate())
    {
        authenticated.POST("/auth/refresh", authHandler.RefreshToken)
        authenticated.POST("/auth/logout", authHandler.Logout)
        authenticated.GET("/me", userHandler.GetCurrentUser)
    }

    // Admin only routes
    admin := router.Group("/api/v1/admin")
    admin.Use(authMiddleware.Authenticate())
    admin.Use(rbacMiddleware.RequireRole("admin"))
    {
        admin.GET("/users", userHandler.ListUsers)
        admin.DELETE("/users/:id", userHandler.DeleteUser)
    }

    // Permission-based routes
    users := router.Group("/api/v1/users")
    users.Use(authMiddleware.Authenticate())
    {
        // Requer permissão users.read
        users.GET("/:id", rbacMiddleware.RequirePermission("users.read"), userHandler.GetUser)

        // Requer permissão users.write
        users.PUT("/:id", rbacMiddleware.RequirePermission("users.write"), userHandler.UpdateUser)

        // Requer permissão users.delete
        users.DELETE("/:id", rbacMiddleware.RequirePermission("users.delete"), userHandler.DeleteUser)
    }
}
```

---

## 8. Helpers

### 8.1 Extract User from Context

```go
// internal/pkg/auth/context.go
package auth

import (
    "errors"
    "github.com/gin-gonic/gin"
)

var (
    ErrUnauthorized = errors.New("unauthorized")
)

// GetUserID extrai user ID do contexto
func GetUserID(c *gin.Context) (string, error) {
    userID, exists := c.Get("user_id")
    if !exists {
        return "", ErrUnauthorized
    }
    return userID.(string), nil
}

// MustGetUserID extrai user ID e panic se não existir
func MustGetUserID(c *gin.Context) string {
    userID, err := GetUserID(c)
    if err != nil {
        panic(err)
    }
    return userID
}

// GetUserRole extrai role do contexto
func GetUserRole(c *gin.Context) (string, error) {
    role, exists := c.Get("user_role")
    if !exists {
        return "", ErrUnauthorized
    }
    return role.(string), nil
}

// GetUserPermissions extrai permissões do contexto
func GetUserPermissions(c *gin.Context) ([]string, error) {
    permissions, exists := c.Get("user_permissions")
    if !exists {
        return nil, ErrUnauthorized
    }
    return permissions.([]string), nil
}

// HasPermission verifica se usuário tem permissão
func HasPermission(c *gin.Context, permission string) bool {
    permissions, err := GetUserPermissions(c)
    if err != nil {
        return false
    }

    for _, p := range permissions {
        if p == permission {
            return true
        }
    }
    return false
}
```

---

## 9. Testes

### 9.1 Mock JWT Manager

```go
// tests/mocks/jwt_manager_mock.go
package mocks

import (
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/infrastructure/auth"
)

type MockJWTManager struct {
    GenerateAccessTokenFunc  func(*entities.User) (string, error)
    ValidateAccessTokenFunc  func(string) (*auth.AccessTokenClaims, error)
}

func (m *MockJWTManager) GenerateAccessToken(user *entities.User) (string, error) {
    if m.GenerateAccessTokenFunc != nil {
        return m.GenerateAccessTokenFunc(user)
    }
    return "mock_access_token", nil
}

func (m *MockJWTManager) ValidateAccessToken(token string) (*auth.AccessTokenClaims, error) {
    if m.ValidateAccessTokenFunc != nil {
        return m.ValidateAccessTokenFunc(token)
    }
    return &auth.AccessTokenClaims{
        UserID: "user_123",
        Email:  "user@example.com",
        Role:   "user",
    }, nil
}
```

### 9.2 Integration Test

```go
// tests/integration/auth_test.go
package integration

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "avantpro-backend/internal/services"
)

func TestLogin(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    userRepo := postgres.NewUserRepository(db)
    jwtManager := auth.NewJWTManager("test_secret")
    refreshTokenStore := auth.NewRefreshTokenStore(setupTestRedis(t))
    authService := services.NewAuthService(userRepo, jwtManager, refreshTokenStore)

    // Create test user
    createTestUser(t, userRepo, "test@example.com", "password123")

    // Test login
    result, err := authService.Login(context.Background(), "test@example.com", "password123")

    assert.NoError(t, err)
    assert.NotEmpty(t, result.AccessToken)
    assert.NotEmpty(t, result.RefreshToken)
    assert.Equal(t, "test@example.com", result.User.Email.String())
}
```

---

## 10. Security Best Practices

### 10.1 Checklist

- ✅ Usar HTTPS em produção
- ✅ Access tokens curtos (15 min)
- ✅ Refresh tokens longos mas revogáveis (7 dias)
- ✅ Armazenar refresh tokens em Redis
- ✅ Implementar logout (revogar refresh tokens)
- ✅ Rate limiting em endpoints de auth
- ✅ CORS configurado corretamente
- ✅ Passwords com bcrypt (cost 12+)
- ✅ Validar state em OAuth flows
- ✅ Não expor informações sensíveis em JWT
- ✅ Logging de tentativas de login falhadas
- ✅ Account lockout após múltiplas tentativas

### 10.2 Environment Variables

```bash
# .env
JWT_SECRET=your-super-secret-key-min-32-chars
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret

OAUTH_REDIRECT_URL=http://localhost:8080

REDIS_URL=redis://localhost:6379
```

---

**Versão**: 2.0
**Última Atualização**: 05/11/2025
