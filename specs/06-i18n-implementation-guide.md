# i18n Implementation Guide - Step-by-Step

**Projeto**: AvantPro Backend
**Versão**: 1.0
**Data**: 05/11/2025

---

## Overview

This guide provides detailed, step-by-step instructions for implementing the i18n system in the AvantPro backend. Follow these steps in order.

**Estimated Time**: 4-6 hours for complete implementation

---

## Phase 1: Core i18n Service Implementation

### Step 1.1: Create i18n Service

**File**: `internal/infrastructure/i18n/i18n.go`

```go
package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"
)

// Service provides internationalization capabilities
type Service interface {
	// T translates a message key for the given language
	// Returns the key itself if translation not found
	T(lang, messageID string, params map[string]interface{}) string

	// SupportedLanguages returns list of available languages
	SupportedLanguages() []string

	// DefaultLanguage returns the default fallback language
	DefaultLanguage() string
}

type service struct {
	translations map[string]map[string]string // lang -> key -> value
	mu           sync.RWMutex                  // protects translations
	defaultLang  string
}

// NewService creates a new i18n service by loading translation files
func NewService(localesDir, defaultLang string) (Service, error) {
	s := &service{
		translations: make(map[string]map[string]string),
		defaultLang:  defaultLang,
	}

	// Load all JSON files from locales directory
	files, err := filepath.Glob(filepath.Join(localesDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read locales directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no translation files found in %s", localesDir)
	}

	for _, file := range files {
		if err := s.loadTranslationFile(file); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
	}

	// Validate default language exists
	if _, ok := s.translations[defaultLang]; !ok {
		return nil, fmt.Errorf("default language %s not found", defaultLang)
	}

	return s, nil
}

func (s *service) loadTranslationFile(path string) error {
	// Extract language code from filename (e.g., "en.json" -> "en")
	lang := filepath.Base(path)
	lang = lang[:len(lang)-len(filepath.Ext(path))] // remove extension

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse JSON
	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return err
	}

	s.translations[lang] = translations
	return nil
}

// T translates a message with optional parameter interpolation
func (s *service) T(lang, messageID string, params map[string]interface{}) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try requested language
	if msg := s.lookup(lang, messageID); msg != "" {
		return s.interpolate(msg, params)
	}

	// Fallback to default language
	if lang != s.defaultLang {
		if msg := s.lookup(s.defaultLang, messageID); msg != "" {
			return s.interpolate(msg, params)
		}
	}

	// Fallback to key itself
	return messageID
}

func (s *service) lookup(lang, messageID string) string {
	if translations, ok := s.translations[lang]; ok {
		if msg, ok := translations[messageID]; ok {
			return msg
		}
	}
	return ""
}

func (s *service) interpolate(msg string, params map[string]interface{}) string {
	if params == nil || len(params) == 0 {
		return msg
	}

	tmpl, err := template.New("msg").Parse(msg)
	if err != nil {
		// Return original message if template parsing fails
		return msg
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		// Return original message if execution fails
		return msg
	}

	return buf.String()
}

func (s *service) SupportedLanguages() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	langs := make([]string, 0, len(s.translations))
	for lang := range s.translations {
		langs = append(langs, lang)
	}
	return langs
}

func (s *service) DefaultLanguage() string {
	return s.defaultLang
}
```

### Step 1.2: Create i18n Service Tests

**File**: `internal/infrastructure/i18n/i18n_test.go`

```go
package i18n

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	service, err := NewService("./locales", "en")
	require.NoError(t, err)
	assert.NotNil(t, service)

	// Should support at least en, pt-BR, es
	langs := service.SupportedLanguages()
	assert.Contains(t, langs, "en")
	assert.Contains(t, langs, "pt-BR")
	assert.Contains(t, langs, "es")
}

func TestNewService_InvalidDirectory(t *testing.T) {
	_, err := NewService("/invalid/path", "en")
	assert.Error(t, err)
}

func TestNewService_MissingDefaultLanguage(t *testing.T) {
	// Create temp directory with only pt-BR
	tmpDir := t.TempDir()
	createTestTranslationFile(t, tmpDir, "pt-BR.json", map[string]string{
		"test": "teste",
	})

	// Should fail because default language "en" doesn't exist
	_, err := NewService(tmpDir, "en")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default language en not found")
}

func TestService_T_Success(t *testing.T) {
	service := createTestService(t)

	result := service.T("pt-BR", "error.user_not_found", nil)
	assert.Equal(t, "Usuário não encontrado", result)

	result = service.T("en", "error.user_not_found", nil)
	assert.Equal(t, "User not found", result)
}

func TestService_T_FallbackToDefault(t *testing.T) {
	service := createTestService(t)

	// Request unsupported language, should fallback to en
	result := service.T("fr", "error.user_not_found", nil)
	assert.Equal(t, "User not found", result)
}

func TestService_T_MissingKey(t *testing.T) {
	service := createTestService(t)

	// Non-existent key should return the key itself
	result := service.T("en", "error.nonexistent", nil)
	assert.Equal(t, "error.nonexistent", result)
}

func TestService_T_ParameterInterpolation(t *testing.T) {
	service := createTestService(t)

	result := service.T("en", "welcome", map[string]interface{}{
		"Name": "John",
	})
	assert.Equal(t, "Welcome, John!", result)

	result = service.T("pt-BR", "welcome", map[string]interface{}{
		"Name": "João",
	})
	assert.Equal(t, "Bem-vindo, João!", result)
}

func TestService_T_InvalidTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	createTestTranslationFile(t, tmpDir, "en.json", map[string]string{
		"invalid": "Hello {{.Name}",
	})

	service, err := NewService(tmpDir, "en")
	require.NoError(t, err)

	// Should return original message even if template is invalid
	result := service.T("en", "invalid", map[string]interface{}{"Name": "Test"})
	assert.Equal(t, "Hello {{.Name}", result)
}

func TestService_T_ThreadSafety(t *testing.T) {
	service := createTestService(t)

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			service.T("pt-BR", "error.user_not_found", nil)
			service.T("en", "welcome", map[string]interface{}{"Name": "Test"})
		}()
	}

	wg.Wait()
	// If no race conditions, test passes
}

func TestService_SupportedLanguages(t *testing.T) {
	service := createTestService(t)

	langs := service.SupportedLanguages()
	assert.Len(t, langs, 3) // en, pt-BR, es
	assert.Contains(t, langs, "en")
	assert.Contains(t, langs, "pt-BR")
	assert.Contains(t, langs, "es")
}

func TestService_DefaultLanguage(t *testing.T) {
	service := createTestService(t)

	assert.Equal(t, "en", service.DefaultLanguage())
}

// Helper functions

func createTestService(t *testing.T) Service {
	// Use actual locales directory from project
	service, err := NewService("./locales", "en")
	require.NoError(t, err)
	return service
}

func createTestTranslationFile(t *testing.T, dir, filename string, translations map[string]string) {
	data, err := json.Marshal(translations)
	require.NoError(t, err)

	path := filepath.Join(dir, filename)
	err = os.WriteFile(path, data, 0644)
	require.NoError(t, err)
}
```

---

## Phase 2: Middleware Implementation

### Step 2.1: Create i18n Middleware

**File**: `internal/handlers/middleware/i18n.go`

```go
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
)

// I18nMiddleware handles language detection and i18n setup
type I18nMiddleware struct {
	i18nService i18n.Service
}

// NewI18nMiddleware creates a new i18n middleware
func NewI18nMiddleware(i18nService i18n.Service) *I18nMiddleware {
	return &I18nMiddleware{
		i18nService: i18nService,
	}
}

// DetectLanguage is a Gin middleware that detects the user's preferred language
// and stores it in the context
func (m *I18nMiddleware) DetectLanguage() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := m.detectLanguage(c)

		// Store in context for handlers to use
		c.Set("lang", lang)
		c.Set("i18n_service", m.i18nService)

		c.Next()
	}
}

// detectLanguage determines the user's preferred language from:
// 1. Query parameter (?lang=pt-BR)
// 2. Accept-Language HTTP header
// 3. Default language (en)
func (m *I18nMiddleware) detectLanguage(c *gin.Context) string {
	// Priority 1: Query parameter
	if lang := c.Query("lang"); lang != "" {
		if m.isSupportedLanguage(lang) {
			return lang
		}
	}

	// Priority 2: Accept-Language header
	if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
		if lang := m.parseAcceptLanguage(acceptLang); lang != "" {
			return lang
		}
	}

	// Priority 3: Default
	return m.i18nService.DefaultLanguage()
}

// parseAcceptLanguage parses the Accept-Language header
// Example: "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7"
// Returns the first supported language in order of preference
func (m *I18nMiddleware) parseAcceptLanguage(header string) string {
	// Split by comma
	languages := strings.Split(header, ",")

	for _, lang := range languages {
		// Remove quality factor: "pt-BR;q=0.9" -> "pt-BR"
		langCode := strings.TrimSpace(strings.Split(lang, ";")[0])

		// Try exact match
		if m.isSupportedLanguage(langCode) {
			return langCode
		}

		// Try base language: "pt-BR" -> "pt"
		if base := m.getBaseLanguage(langCode); m.isSupportedLanguage(base) {
			return base
		}
	}

	return ""
}

// isSupportedLanguage checks if a language is supported
func (m *I18nMiddleware) isSupportedLanguage(lang string) bool {
	supported := m.i18nService.SupportedLanguages()
	for _, sl := range supported {
		if sl == lang {
			return true
		}
	}
	return false
}

// getBaseLanguage extracts the base language from a locale
// Examples: "pt-BR" -> "pt", "en-US" -> "en", "es" -> "es"
func (m *I18nMiddleware) getBaseLanguage(locale string) string {
	parts := strings.Split(locale, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return locale
}
```

### Step 2.2: Create Middleware Tests

**File**: `internal/handlers/middleware/i18n_test.go`

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
)

func TestI18nMiddleware_DetectLanguage_QueryParameter(t *testing.T) {
	middleware, router := setupTestMiddleware(t)

	router.GET("/test", middleware.DetectLanguage(), func(c *gin.Context) {
		lang := c.GetString("lang")
		c.JSON(200, gin.H{"lang": lang})
	})

	// Test with query parameter
	req := httptest.NewRequest("GET", "/test?lang=pt-BR", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"lang":"pt-BR"`)
}

func TestI18nMiddleware_DetectLanguage_AcceptLanguageHeader(t *testing.T) {
	middleware, router := setupTestMiddleware(t)

	router.GET("/test", middleware.DetectLanguage(), func(c *gin.Context) {
		lang := c.GetString("lang")
		c.JSON(200, gin.H{"lang": lang})
	})

	// Test with Accept-Language header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"lang":"pt-BR"`)
}

func TestI18nMiddleware_DetectLanguage_DefaultFallback(t *testing.T) {
	middleware, router := setupTestMiddleware(t)

	router.GET("/test", middleware.DetectLanguage(), func(c *gin.Context) {
		lang := c.GetString("lang")
		c.JSON(200, gin.H{"lang": lang})
	})

	// Test without any language hints
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"lang":"en"`)
}

func TestI18nMiddleware_DetectLanguage_UnsupportedLanguage(t *testing.T) {
	middleware, router := setupTestMiddleware(t)

	router.GET("/test", middleware.DetectLanguage(), func(c *gin.Context) {
		lang := c.GetString("lang")
		c.JSON(200, gin.H{"lang": lang})
	})

	// Test with unsupported language
	req := httptest.NewRequest("GET", "/test?lang=fr", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fallback to default (en)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"lang":"en"`)
}

func TestI18nMiddleware_ParseAcceptLanguage_MultipleLanguages(t *testing.T) {
	middleware, router := setupTestMiddleware(t)

	router.GET("/test", middleware.DetectLanguage(), func(c *gin.Context) {
		lang := c.GetString("lang")
		c.JSON(200, gin.H{"lang": lang})
	})

	// Test with multiple languages, es is first supported
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Language", "fr;q=0.9,es;q=0.8,en;q=0.7")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"lang":"es"`)
}

func TestI18nMiddleware_StoresServiceInContext(t *testing.T) {
	middleware, router := setupTestMiddleware(t)

	router.GET("/test", middleware.DetectLanguage(), func(c *gin.Context) {
		service, exists := c.Get("i18n_service")
		assert.True(t, exists)
		assert.NotNil(t, service)

		// Verify it's the correct type
		_, ok := service.(i18n.Service)
		assert.True(t, ok)

		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

// Helper function
func setupTestMiddleware(t *testing.T) (*I18nMiddleware, *gin.Engine) {
	// Create test i18n service
	service, err := i18n.NewService("../../infrastructure/i18n/locales", "en")
	require.NoError(t, err)

	middleware := NewI18nMiddleware(service)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return middleware, router
}
```

---

## Phase 3: Helper Functions

### Step 3.1: Create i18n Helper

**File**: `internal/handlers/dto/i18n_helper.go`

```go
package dto

import (
	"github.com/gin-gonic/gin"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
)

// T translates a message using the i18n service from context
func T(c *gin.Context, messageID string, params ...map[string]interface{}) string {
	// Get language from context
	lang := c.GetString("lang")
	if lang == "" {
		lang = "en" // fallback
	}

	// Get i18n service from context
	i18nService, exists := c.Get("i18n_service")
	if !exists {
		return messageID // fallback if service not available
	}

	// Prepare template data
	var templateData map[string]interface{}
	if len(params) > 0 {
		templateData = params[0]
	}

	// Translate
	return i18nService.(i18n.Service).T(lang, messageID, templateData)
}

// GetLanguage returns the current request's language
func GetLanguage(c *gin.Context) string {
	if lang := c.GetString("lang"); lang != "" {
		return lang
	}
	return "en"
}
```

### Step 3.2: Update ErrorResponse DTOs

**File**: Update `internal/handlers/dto/common.go`

Add these new functions (don't remove existing ones):

```go
// Add to existing common.go file

// NewErrorResponseI18n creates RFC 7807 error response with i18n support
func NewErrorResponseI18n(c *gin.Context, problemType, titleKey string, status int, detailKey string, params ...map[string]interface{}) ErrorResponse {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	var templateData map[string]interface{}
	if len(params) > 0 {
		templateData = params[0]
	}

	return ErrorResponse{
		Type:     baseURL + problemType,
		Title:    T(c, titleKey),
		Status:   status,
		Detail:   T(c, detailKey, templateData),
		Instance: c.Request.URL.Path,
	}
}

// ValidationErrorResponseI18n creates validation error response with i18n
func ValidationErrorResponseI18n(c *gin.Context, errors []ValidationError) ErrorResponse {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return ErrorResponse{
		Type:     baseURL + "/problems/validation-error",
		Title:    T(c, "error.validation.title"),
		Status:   http.StatusBadRequest,
		Detail:   T(c, "error.validation.detail"),
		Instance: c.Request.URL.Path,
		Errors:   errors,
	}
}

// NotFoundErrorResponseI18n creates not found error response with i18n
func NotFoundErrorResponseI18n(c *gin.Context, resource string) ErrorResponse {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return ErrorResponse{
		Type:     baseURL + "/problems/not-found",
		Title:    T(c, "error.not_found.title"),
		Status:   http.StatusNotFound,
		Detail:   T(c, "error.not_found.detail", map[string]interface{}{"Resource": resource}),
		Instance: c.Request.URL.Path,
	}
}

// ConflictErrorResponseI18n creates conflict error response with i18n
func ConflictErrorResponseI18n(c *gin.Context, detailKey string, params ...map[string]interface{}) ErrorResponse {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	var templateData map[string]interface{}
	if len(params) > 0 {
		templateData = params[0]
	}

	return ErrorResponse{
		Type:     baseURL + "/problems/conflict",
		Title:    T(c, "error.conflict.title"),
		Status:   http.StatusConflict,
		Detail:   T(c, detailKey, templateData),
		Instance: c.Request.URL.Path,
	}
}

// InternalErrorResponseI18n creates internal error response with i18n
func InternalErrorResponseI18n(c *gin.Context) ErrorResponse {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return ErrorResponse{
		Type:     baseURL + "/problems/internal-error",
		Title:    T(c, "error.internal.title"),
		Status:   http.StatusInternalServerError,
		Detail:   T(c, "error.internal.detail"),
		Instance: c.Request.URL.Path,
	}
}

// UnauthorizedErrorResponseI18n creates unauthorized error response with i18n
func UnauthorizedErrorResponseI18n(c *gin.Context) ErrorResponse {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return ErrorResponse{
		Type:     baseURL + "/problems/unauthorized",
		Title:    T(c, "error.unauthorized.title"),
		Status:   http.StatusUnauthorized,
		Detail:   T(c, "error.unauthorized.detail"),
		Instance: c.Request.URL.Path,
	}
}

// ForbiddenErrorResponseI18n creates forbidden error response with i18n
func ForbiddenErrorResponseI18n(c *gin.Context) ErrorResponse {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return ErrorResponse{
		Type:     baseURL + "/problems/forbidden",
		Title:    T(c, "error.forbidden.title"),
		Status:   http.StatusForbidden,
		Detail:   T(c, "error.forbidden.detail"),
		Instance: c.Request.URL.Path,
	}
}
```

---

## Phase 4: Main Application Integration

### Step 4.1: Update main.go

**File**: `cmd/api/main.go`

Add these changes:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rafabene/avantpro-backend/internal/handlers/http"
	"github.com/rafabene/avantpro-backend/internal/handlers/middleware"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/config"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"  // NEW
	"github.com/rafabene/avantpro-backend/internal/infrastructure/logging"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/persistence/postgres"
	"github.com/rafabene/avantpro-backend/internal/services"
)

func main() {
	// Carregar configurações
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Inicializar logger
	logger := logging.NewSlogLogger(cfg.Logging.Level)
	logger.Info("starting avantpro backend",
		"env", cfg.Env,
		"version", "dev",
	)

	// NEW: Inicializar i18n service
	i18nService, err := i18n.NewService(
		"./internal/infrastructure/i18n/locales",
		"en",
	)
	if err != nil {
		logger.Error("failed to initialize i18n service", "error", err)
		log.Fatal(err)
	}
	logger.Info("i18n service initialized",
		"languages", i18nService.SupportedLanguages(),
		"default", i18nService.DefaultLanguage(),
	)

	// Conectar ao banco de dados
	db, err := postgres.NewDatabaseConnection(&cfg.Database, logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		log.Fatal(err)
	}

	// Inicializar repositories
	userRepo := postgres.NewUserRepository(db)
	uow := postgres.NewUnitOfWork(db)

	// Inicializar services
	userService := services.NewUserService(userRepo, uow, logger)

	// Inicializar handlers
	userHandler := http.NewUserHandler(userService)

	// Setup Gin
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware global para adicionar base URL ao contexto
	router.Use(func(c *gin.Context) {
		c.Set("base_url", cfg.Server.BaseURL)
		c.Next()
	})

	// NEW: i18n middleware (must come early in chain)
	i18nMiddleware := middleware.NewI18nMiddleware(i18nService)
	router.Use(i18nMiddleware.DetectLanguage())

	// Middleware CORS
	router.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"env":    cfg.Env,
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Users
		users := v1.Group("/users")
		{
			users.POST("", userHandler.CreateUser)
			users.GET("/:id", userHandler.GetUser)
			users.GET("", userHandler.ListUsers)
		}
	}

	// HTTP Server
	srv := &http.Server{
		Addr:    cfg.Server.Host + ":" + cfg.Server.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		logger.Info("server starting",
			"host", cfg.Server.Host,
			"port", cfg.Server.Port,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		log.Fatal(err)
	}

	logger.Info("server exited")
}
```

---

## Phase 5: Handler Updates

### Step 5.1: Update User Handler

**File**: `internal/handlers/http/user_handler.go`

Replace the existing handler methods with i18n-aware versions:

```go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rafabene/avantpro-backend/internal/domain/errors"
	"github.com/rafabene/avantpro-backend/internal/handlers/dto"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// UserHandler lida com requisições HTTP relacionadas a usuários
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler cria um novo UserHandler
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// CreateUser cria um novo usuário
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response := dto.NewErrorResponseI18n(
			c,
			errors.ProblemTypeValidation,
			"error.validation.title",
			http.StatusBadRequest,
			"error.validation.detail",
		)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// TODO: Implementar lógica completa
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": dto.T(c, "user_created"),
	})
}

// GetUser busca um usuário por ID
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		if err == errors.ErrUserNotFound {
			response := dto.NotFoundErrorResponseI18n(c, "user")
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := dto.InternalErrorResponseI18n(c)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	c.JSON(http.StatusOK, dto.ToUserResponse(user))
}

// ListUsers lista usuários
func (h *UserHandler) ListUsers(c *gin.Context) {
	// TODO: Implementar paginação e filtros
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "List users endpoint - implementation pending",
	})
}
```

---

## Phase 6: Testing

### Step 6.1: Manual Testing

Test the i18n system with curl:

```bash
# Test English (default)
curl http://localhost:8080/api/v1/users/999

# Test Portuguese (Accept-Language header)
curl -H "Accept-Language: pt-BR" http://localhost:8080/api/v1/users/999

# Test Spanish (query parameter)
curl http://localhost:8080/api/v1/users/999?lang=es

# Test parameter interpolation
curl http://localhost:8080/health?lang=pt-BR
```

Expected responses:

**English**:
```json
{
  "type": "http://localhost:8080/problems/not-found",
  "title": "Resource Not Found",
  "status": 404,
  "detail": "user not found",
  "instance": "/api/v1/users/999"
}
```

**Portuguese**:
```json
{
  "type": "http://localhost:8080/problems/not-found",
  "title": "Recurso Não Encontrado",
  "status": 404,
  "detail": "user não encontrado",
  "instance": "/api/v1/users/999"
}
```

### Step 6.2: Run Unit Tests

```bash
# Test i18n service
go test ./internal/infrastructure/i18n -v

# Test i18n middleware
go test ./internal/handlers/middleware -v -run TestI18n

# Run all tests
go test ./... -v
```

---

## Phase 7: Validation

### Step 7.1: Validation Checklist

- [ ] i18n service loads all translation files
- [ ] i18n service handles missing translations gracefully
- [ ] i18n service interpolates parameters correctly
- [ ] Middleware detects language from query parameter
- [ ] Middleware detects language from Accept-Language header
- [ ] Middleware falls back to default language
- [ ] Helper functions work in handlers
- [ ] Error responses are translated
- [ ] Thread safety (no race conditions)
- [ ] All tests pass

### Step 7.2: Integration Test

Create an integration test to verify the complete flow:

**File**: `tests/integration/i18n_test.go`

```go
package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rafabene/avantpro-backend/internal/handlers/dto"
	"github.com/rafabene/avantpro-backend/internal/handlers/middleware"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
)

func TestI18n_EndToEnd(t *testing.T) {
	// Setup
	i18nService, err := i18n.NewService("../../internal/infrastructure/i18n/locales", "en")
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	i18nMiddleware := middleware.NewI18nMiddleware(i18nService)
	router.Use(i18nMiddleware.DetectLanguage())

	// Test endpoint
	router.GET("/test/not-found", func(c *gin.Context) {
		response := dto.NotFoundErrorResponseI18n(c, "user")
		c.JSON(http.StatusNotFound, response)
	})

	// Test 1: English (default)
	t.Run("English_Default", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/not-found", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Resource Not Found")
		assert.Contains(t, w.Body.String(), "user not found")
	})

	// Test 2: Portuguese
	t.Run("Portuguese_AcceptLanguage", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/not-found", nil)
		req.Header.Set("Accept-Language", "pt-BR")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Recurso Não Encontrado")
		assert.Contains(t, w.Body.String(), "não encontrado")
	})

	// Test 3: Spanish (query param)
	t.Run("Spanish_QueryParam", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/not-found?lang=es", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Recurso No Encontrado")
		assert.Contains(t, w.Body.String(), "no encontrado")
	})
}
```

---

## Troubleshooting

### Issue 1: Translation files not found

**Error**: `no translation files found in ./internal/infrastructure/i18n/locales`

**Solution**:
- Verify the path is correct relative to where you run the application
- Use absolute path or make path configurable
- Check file permissions

### Issue 2: Race condition detected

**Error**: `WARNING: DATA RACE`

**Solution**:
- Ensure all access to `translations` map uses `sync.RWMutex`
- Use `RLock()` for reads, `Lock()` for writes
- Never modify translations after initialization

### Issue 3: Template parsing fails

**Error**: Messages with `{{.Name}}` not interpolating

**Solution**:
- Check template syntax in JSON files
- Ensure parameter names match (case-sensitive)
- Verify parameters are passed correctly

---

## Summary

After completing all phases, you will have:

1. A thread-safe i18n service that loads translations from JSON files
2. Middleware that automatically detects user language preferences
3. Helper functions for easy translation in handlers
4. i18n-aware error responses (RFC 7807 compliant)
5. Complete test coverage
6. Integration with existing Clean Architecture patterns

**Next Steps**:
1. Add more translation keys as needed
2. Consider adding pluralization support
3. Add translation validation to CI/CD
4. Create admin interface for managing translations

---

**Version**: 1.0
**Last Updated**: 05/11/2025
**Status**: Implementation Guide Complete
