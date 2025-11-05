# i18n (Internationalization) System - Architecture & Implementation Specification

**Projeto**: AvantPro Backend
**Versão**: 1.0
**Data**: 05/11/2025
**Status**: Implementation Ready

---

## 1. Executive Summary

### 1.1 Current State
- Translation files exist in `internal/infrastructure/i18n/locales/` (en.json, pt-BR.json, es.json)
- Error codes in `internal/domain/errors/errors.go` use i18n keys (e.g., "error.user_not_found")
- **MISSING**: i18n service, middleware, and integration

### 1.2 Required Implementation
This specification defines a complete i18n system that:
1. Loads and manages translations from JSON files
2. Detects user language preferences from HTTP headers/query parameters
3. Provides thread-safe translation services for concurrent requests
4. Integrates with existing error handling (RFC 7807)
5. Supports parameter interpolation (e.g., {{.Name}}, {{.Email}})
6. Falls back to English when translations are missing

### 1.3 Design Goals
- **Minimal Dependencies**: Use standard library where possible, lightweight packages only
- **Thread-Safe**: Support concurrent requests without race conditions
- **Clean Architecture**: Follow existing project patterns (hexagonal/clean architecture)
- **Dependency Injection**: Wire-compatible, testable components
- **Performance**: Load translations once at startup, cache in memory
- **Flexibility**: Easy to add new languages and translation keys

---

## 2. Architecture Overview

### 2.1 Component Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                      HTTP Request                             │
│                 Accept-Language: pt-BR                        │
│                 ?lang=es (optional)                           │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│              Language Detection Middleware                    │
│  • Parse Accept-Language header                              │
│  • Check ?lang query parameter                               │
│  • Store language in Gin context                             │
│  • Set "lang" and "i18n_service" in context                  │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                     Handler Layer                             │
│  • Extract i18n service from context                         │
│  • Call T(ctx, "message.key", params)                        │
│  • Use translated messages in responses                      │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                   I18n Service (Core)                         │
│  • Thread-safe translation lookup                            │
│  • Parameter interpolation ({{.Name}})                       │
│  • Fallback to default language (en)                         │
│  • Pluralization support (future)                            │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                Translation Store (In-Memory)                  │
│  map[string]map[string]string                                │
│  {"en": {...}, "pt-BR": {...}, "es": {...}}                  │
│  Loaded once at application startup                          │
└──────────────────────────────────────────────────────────────┘
```

### 2.2 Data Flow

```
1. HTTP Request arrives with Accept-Language: pt-BR
   ↓
2. I18nMiddleware detects language → stores "pt-BR" in context
   ↓
3. Handler receives request
   ↓
4. Handler calls i18nHelper.T(c, "error.user_not_found")
   ↓
5. I18nService looks up translation:
   - Check pt-BR translations → "Usuário não encontrado"
   - If not found → fallback to en → "User not found"
   - If still not found → return key "error.user_not_found"
   ↓
6. Return translated string to handler
   ↓
7. Handler includes translated message in response
```

---

## 3. Component Specifications

### 3.1 I18n Service (Core Translation Engine)

**Location**: `internal/infrastructure/i18n/i18n.go`

**Responsibilities**:
- Load translation files at startup
- Provide thread-safe translation lookup
- Support parameter interpolation using Go templates
- Fallback to default language when translation missing

**Interface**:
```go
package i18n

type Service interface {
    // T translates a message key for the given language
    // Returns the key itself if translation not found
    T(lang, messageID string, params map[string]interface{}) string

    // SupportedLanguages returns list of available languages
    SupportedLanguages() []string

    // DefaultLanguage returns the default fallback language
    DefaultLanguage() string
}
```

**Implementation Requirements**:
1. **Thread Safety**: Use `sync.RWMutex` or immutable data structures
2. **Memory Efficiency**: Load all translations once at startup
3. **Template Support**: Use `text/template` for parameter interpolation
4. **Error Handling**: Never panic, always return fallback (key or default language)
5. **Performance**: O(1) lookup time for translations

**Data Structure**:
```go
type service struct {
    translations map[string]map[string]string // lang -> key -> value
    templates    map[string]*template.Template // cached templates
    mu           sync.RWMutex
    defaultLang  string
}
```

**Key Methods**:
```go
// NewService loads translations from embedded JSON files
func NewService(localesDir string, defaultLang string) (Service, error)

// T translates with parameter interpolation
func (s *service) T(lang, messageID string, params map[string]interface{}) string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Try requested language
    if translations, ok := s.translations[lang]; ok {
        if msg, ok := translations[messageID]; ok {
            return s.interpolate(msg, params)
        }
    }

    // Fallback to default language
    if translations, ok := s.translations[s.defaultLang]; ok {
        if msg, ok := translations[messageID]; ok {
            return s.interpolate(msg, params)
        }
    }

    // Fallback to key itself
    return messageID
}
```

### 3.2 Language Detection Middleware

**Location**: `internal/handlers/middleware/i18n.go`

**Responsibilities**:
- Detect user's preferred language from HTTP request
- Store language preference in Gin context
- Provide i18n service instance to handlers

**Detection Priority** (highest to lowest):
1. Query parameter: `?lang=pt-BR`
2. Accept-Language HTTP header: `Accept-Language: pt-BR,pt;q=0.9,en;q=0.8`
3. Default language: `en`

**Implementation**:
```go
package middleware

type I18nMiddleware struct {
    i18nService i18n.Service
}

func NewI18nMiddleware(i18nService i18n.Service) *I18nMiddleware {
    return &I18nMiddleware{i18nService: i18nService}
}

func (m *I18nMiddleware) DetectLanguage() gin.HandlerFunc {
    return func(c *gin.Context) {
        lang := m.detectLanguage(c)

        // Store in context for handlers
        c.Set("lang", lang)
        c.Set("i18n_service", m.i18nService)

        c.Next()
    }
}

func (m *I18nMiddleware) detectLanguage(c *gin.Context) string {
    // 1. Check query parameter
    if lang := c.Query("lang"); lang != "" {
        if m.isSupportedLanguage(lang) {
            return lang
        }
    }

    // 2. Parse Accept-Language header
    if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
        if lang := m.parseAcceptLanguage(acceptLang); lang != "" {
            return lang
        }
    }

    // 3. Default
    return m.i18nService.DefaultLanguage()
}
```

**Accept-Language Parsing**:
```go
// parseAcceptLanguage parses "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7"
// Returns first supported language in order of preference
func (m *I18nMiddleware) parseAcceptLanguage(header string) string {
    // Split by comma
    languages := strings.Split(header, ",")

    for _, lang := range languages {
        // Remove quality factor: "pt-BR;q=0.9" -> "pt-BR"
        langCode := strings.TrimSpace(strings.Split(lang, ";")[0])

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
```

### 3.3 Translation Helper Functions

**Location**: `internal/handlers/dto/i18n_helper.go`

**Purpose**: Provide convenient helper functions for handlers

```go
package dto

import (
    "github.com/gin-gonic/gin"
    "github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
)

// T translates a message using the i18n service from context
func T(c *gin.Context, messageID string, params ...map[string]interface{}) string {
    lang := c.GetString("lang")
    if lang == "" {
        lang = "en"
    }

    i18nService, exists := c.Get("i18n_service")
    if !exists {
        return messageID
    }

    var templateData map[string]interface{}
    if len(params) > 0 {
        templateData = params[0]
    }

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

### 3.4 Updated ErrorResponse DTOs

**Location**: `internal/handlers/dto/common.go`

**Changes Required**:
- Update `NewErrorResponse` to use i18n service
- Add helper functions that accept Gin context for translation

```go
// NewErrorResponseI18n creates RFC 7807 error with i18n support
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

// Pre-built error response helpers
func ValidationErrorResponseI18n(c *gin.Context, errors []ValidationError) ErrorResponse {
    return NewErrorResponseI18n(
        c,
        errors.ProblemTypeValidation,
        "error.validation.title",
        http.StatusBadRequest,
        "error.validation.detail",
    ).WithErrors(errors)
}

func NotFoundErrorResponseI18n(c *gin.Context, resourceKey string) ErrorResponse {
    return NewErrorResponseI18n(
        c,
        errors.ProblemTypeNotFound,
        "error.not_found.title",
        http.StatusNotFound,
        "error.not_found.detail",
        map[string]interface{}{"Resource": T(c, resourceKey)},
    )
}

// Add WithErrors method to ErrorResponse
func (e ErrorResponse) WithErrors(errors []ValidationError) ErrorResponse {
    e.Errors = errors
    return e
}
```

---

## 4. Integration Points

### 4.1 Main Application Setup

**File**: `cmd/api/main.go`

**Required Changes**:
```go
package main

import (
    // ... existing imports
    "github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
    "github.com/rafabene/avantpro-backend/internal/handlers/middleware"
)

func main() {
    // ... existing config and logger setup

    // Initialize i18n service
    i18nService, err := i18n.NewService(
        "./internal/infrastructure/i18n/locales",
        "en", // default language
    )
    if err != nil {
        logger.Error("failed to initialize i18n service", "error", err)
        log.Fatal(err)
    }
    logger.Info("i18n service initialized", "languages", i18nService.SupportedLanguages())

    // ... existing database and repository setup

    // ... existing service initialization

    // ... existing handler initialization

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

    // ... rest of setup
}
```

### 4.2 Handler Integration Example

**File**: `internal/handlers/http/user_handler.go`

**Before** (hardcoded English):
```go
func (h *UserHandler) GetUser(c *gin.Context) {
    id := c.Param("id")

    user, err := h.userService.GetUser(c.Request.Context(), id)
    if err != nil {
        if err == errors.ErrUserNotFound {
            response := dto.NewErrorResponse(c, errors.ProblemTypeNotFound, "Not Found", http.StatusNotFound, "User not found")
            c.JSON(http.StatusNotFound, response)
            return
        }
        response := dto.NewErrorResponse(c, errors.ProblemTypeInternal, "Internal Error", http.StatusInternalServerError, "An unexpected error occurred")
        c.JSON(http.StatusInternalServerError, response)
        return
    }

    c.JSON(http.StatusOK, dto.ToUserResponse(user))
}
```

**After** (with i18n):
```go
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
```

### 4.3 Validation Error Integration

**File**: `internal/handlers/http/user_handler.go`

```go
func (h *UserHandler) CreateUser(c *gin.Context) {
    var req dto.CreateUserRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        // Format validation errors with i18n
        validationErrors := formatValidationErrorsI18n(c, err)

        response := dto.ValidationErrorResponseI18n(c, validationErrors)
        c.JSON(http.StatusBadRequest, response)
        return
    }

    // ... rest of handler
}

// formatValidationErrorsI18n formats validator errors with translated messages
func formatValidationErrorsI18n(c *gin.Context, err error) []dto.ValidationError {
    if validationErrors, ok := err.(validator.ValidationErrors); ok {
        errors := make([]dto.ValidationError, 0, len(validationErrors))

        for _, e := range validationErrors {
            errors = append(errors, dto.ValidationError{
                Field:   e.Field(),
                Tag:     e.Tag(),
                Value:   fmt.Sprintf("%v", e.Value()),
                Message: getValidationMessageI18n(c, e),
            })
        }

        return errors
    }

    return nil
}

func getValidationMessageI18n(c *gin.Context, e validator.FieldError) string {
    // Map validator tag to translation key
    messageKey := fmt.Sprintf("validation_%s", e.Tag())

    params := map[string]interface{}{
        "Field": e.Field(),
        "Min":   e.Param(),
        "Max":   e.Param(),
    }

    return dto.T(c, messageKey, params)
}
```

---

## 5. Translation File Management

### 5.1 File Structure

```
internal/infrastructure/i18n/locales/
├── en.json      (English - Default)
├── pt-BR.json   (Brazilian Portuguese)
└── es.json      (Spanish)
```

### 5.2 Translation Key Naming Conventions

**Pattern**: `category.subcategory.key`

**Categories**:
- `error.*` - Error messages
- `validation_*` - Validation messages
- `success.*` - Success messages
- `common.*` - Common UI labels
- `email.*` - Email templates

**Examples**:
```json
{
  "error.user_not_found": "User not found",
  "error.validation.title": "Validation Failed",
  "validation_required": "{{.Field}} is required",
  "validation_email": "{{.Field}} must be a valid email",
  "success.user_created": "User created successfully"
}
```

### 5.3 Adding New Languages

**Steps**:
1. Create new JSON file: `internal/infrastructure/i18n/locales/fr.json`
2. Copy structure from `en.json`
3. Translate all keys to target language
4. Restart application (translations loaded at startup)

**No code changes required** - the i18n service automatically loads all JSON files in the locales directory.

---

## 6. Implementation Details

### 6.1 Parameter Interpolation

Use Go's `text/template` for parameter substitution:

**Template Syntax**: `{{.ParameterName}}`

**Example**:
```json
{
  "welcome": "Welcome, {{.Name}}!",
  "email_sent": "Email sent to {{.Email}}",
  "error.not_found.detail": "{{.Resource}} not found"
}
```

**Usage**:
```go
msg := i18nService.T("pt-BR", "welcome", map[string]interface{}{
    "Name": "João",
})
// Returns: "Bem-vindo, João!"
```

### 6.2 Thread Safety

**Requirements**:
- Multiple goroutines will access i18n service concurrently (one per HTTP request)
- Translations are read-only after initialization
- Use `sync.RWMutex` for safe concurrent reads

**Implementation**:
```go
type service struct {
    translations map[string]map[string]string
    mu           sync.RWMutex // protects translations map
    defaultLang  string
}

func (s *service) T(lang, key string, params map[string]interface{}) string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // ... safe concurrent reads
}
```

### 6.3 Error Handling

**Principle**: Never fail, always provide fallback

**Fallback Chain**:
1. Try requested language translation
2. Fall back to default language (en)
3. Fall back to message key itself
4. Never return empty string or panic

**Example**:
```go
// Request: lang=pt-BR, key="error.user_not_active"
// pt-BR has no translation for this key

// 1. Check pt-BR: NOT FOUND
// 2. Check en (default): "User account is not active"
// 3. Return: "User account is not active"

// If even en doesn't have it:
// 4. Return: "error.user_not_active" (the key itself)
```

---

## 7. Testing Strategy

### 7.1 Unit Tests

**File**: `internal/infrastructure/i18n/i18n_test.go`

```go
func TestService_T_Success(t *testing.T) {
    service := setupTestService(t)

    result := service.T("pt-BR", "error.user_not_found", nil)
    assert.Equal(t, "Usuário não encontrado", result)
}

func TestService_T_FallbackToDefault(t *testing.T) {
    service := setupTestService(t)

    // Request non-existent language, should fallback to en
    result := service.T("fr", "error.user_not_found", nil)
    assert.Equal(t, "User not found", result)
}

func TestService_T_ParameterInterpolation(t *testing.T) {
    service := setupTestService(t)

    result := service.T("en", "welcome", map[string]interface{}{
        "Name": "John",
    })
    assert.Equal(t, "Welcome, John!", result)
}

func TestService_T_ThreadSafety(t *testing.T) {
    service := setupTestService(t)

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            service.T("pt-BR", "error.user_not_found", nil)
        }()
    }
    wg.Wait()
    // No race conditions should occur
}
```

### 7.2 Integration Tests

**File**: `tests/integration/i18n_handler_test.go`

```go
func TestUserHandler_GetUser_NotFound_I18n(t *testing.T) {
    router := setupTestRouter(t)

    // Test Portuguese
    req := httptest.NewRequest("GET", "/api/v1/users/999", nil)
    req.Header.Set("Accept-Language", "pt-BR")
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusNotFound, w.Code)

    var response dto.ErrorResponse
    json.Unmarshal(w.Body.Bytes(), &response)

    assert.Equal(t, "Recurso Não Encontrado", response.Title)
    assert.Contains(t, response.Detail, "não encontrado")
}
```

---

## 8. Performance Considerations

### 8.1 Memory Usage

**Estimated Memory**:
- 3 languages × ~50 keys × ~50 bytes/value = ~7.5 KB
- Template cache: ~5 KB
- **Total**: ~12-15 KB (negligible)

**Trade-off**: Load all translations at startup (one-time cost) vs. loading on-demand
**Decision**: Pre-load all (better performance, acceptable memory usage)

### 8.2 Lookup Performance

**Data Structure**: `map[string]map[string]string`
**Lookup Complexity**: O(1) for language lookup + O(1) for key lookup = O(1) total
**Concurrency**: Read lock overhead is minimal (RWMutex optimized for many readers)

**Benchmarks** (expected):
```
BenchmarkService_T_NoParams-8      5000000    250 ns/op
BenchmarkService_T_WithParams-8    2000000    650 ns/op
```

---

## 9. Migration Path

### 9.1 Phase 1: Core Implementation (Week 1)
1. Implement i18n.Service with basic translation lookup
2. Implement I18nMiddleware for language detection
3. Create helper functions (T, GetLanguage)
4. Write unit tests

### 9.2 Phase 2: Handler Integration (Week 1-2)
1. Update dto/common.go with i18n-aware error responses
2. Update user_handler.go to use i18n
3. Add validation error i18n support
4. Write integration tests

### 9.3 Phase 3: Complete Migration (Week 2)
1. Update all remaining handlers
2. Add missing translation keys
3. Add Spanish translations (if not complete)
4. End-to-end testing

### 9.4 Backward Compatibility

**Strategy**: Support both old and new error response methods during transition

```go
// Old method (still works)
func NewErrorResponse(c *gin.Context, problemType, title string, status int, detail string) ErrorResponse

// New method (recommended)
func NewErrorResponseI18n(c *gin.Context, problemType, titleKey string, status int, detailKey string, params ...map[string]interface{}) ErrorResponse
```

---

## 10. Configuration

### 10.1 Environment Variables

Add to `.env`:
```bash
# i18n Configuration
I18N_DEFAULT_LANGUAGE=en
I18N_LOCALES_DIR=./internal/infrastructure/i18n/locales
```

### 10.2 Config Structure

Update `internal/infrastructure/config/config.go`:
```go
type Config struct {
    // ... existing fields
    I18n I18nConfig
}

type I18nConfig struct {
    DefaultLanguage string
    LocalesDir      string
}
```

Load in `config.Load()`:
```go
I18n: I18nConfig{
    DefaultLanguage: viper.GetString("I18N_DEFAULT_LANGUAGE"),
    LocalesDir:      viper.GetString("I18N_LOCALES_DIR"),
},
```

---

## 11. Dependencies

### 11.1 Standard Library Only

**Decision**: Use only Go standard library, no external i18n packages

**Rationale**:
- Translation files are simple (no pluralization needed yet)
- Standard `text/template` handles interpolation
- Standard `encoding/json` loads translation files
- Simpler, fewer dependencies, more control

**Future**: If pluralization or complex formatting needed, consider `golang.org/x/text` or `github.com/nicksnyder/go-i18n/v2`

### 11.2 Existing Dependencies (No Changes)
- Gin (already used)
- validator (already used)
- GORM (already used)

---

## 12. Security Considerations

### 12.1 Input Validation

**Language Code Validation**:
```go
func (m *I18nMiddleware) isSupportedLanguage(lang string) bool {
    // Whitelist approach - only allow known languages
    supported := m.i18nService.SupportedLanguages()
    for _, sl := range supported {
        if sl == lang {
            return true
        }
    }
    return false
}
```

**Prevent Injection**:
- Language codes validated against whitelist
- Translation parameters sanitized by Go templates
- No user input directly used as translation keys

### 12.2 Content Security

**Translation Integrity**:
- Translation files stored in application binary (not user-editable)
- No runtime modification of translations
- Immutable after loading

---

## 13. Monitoring & Observability

### 13.1 Logging

**Log i18n events**:
```go
// At startup
logger.Info("i18n service initialized",
    "languages", i18nService.SupportedLanguages(),
    "default", i18nService.DefaultLanguage(),
    "keys_count", totalKeys,
)

// On missing translation (debug level)
logger.Debug("translation not found",
    "lang", lang,
    "key", messageID,
    "fallback", fallbackMsg,
)
```

### 13.2 Metrics (Future)

**Potential Metrics**:
- Language usage distribution (pt-BR: 60%, en: 30%, es: 10%)
- Missing translation requests
- Translation lookup performance

---

## 14. Documentation for Developers

### 14.1 Quick Start Guide

**Adding a new translation**:
1. Add key to all JSON files in `locales/`
2. Restart application
3. Use in code: `dto.T(c, "your.new.key")`

**Using i18n in handlers**:
```go
// Simple translation
message := dto.T(c, "success.user_created")

// With parameters
message := dto.T(c, "welcome", map[string]interface{}{
    "Name": user.Name,
})

// In error responses
response := dto.NotFoundErrorResponseI18n(c, "user")
```

### 14.2 Best Practices

1. **Always use translation keys, never hardcode strings**
   ```go
   // ❌ Bad
   c.JSON(200, gin.H{"message": "User created successfully"})

   // ✅ Good
   c.JSON(200, gin.H{"message": dto.T(c, "success.user_created")})
   ```

2. **Use semantic keys, not English text as keys**
   ```go
   // ❌ Bad
   dto.T(c, "User not found")

   // ✅ Good
   dto.T(c, "error.user_not_found")
   ```

3. **Keep translations in sync across all language files**

4. **Test with different languages in development**
   ```bash
   curl -H "Accept-Language: pt-BR" http://localhost:8080/api/v1/users/1
   ```

---

## 15. Future Enhancements

### 15.1 Potential Improvements

1. **Pluralization Support**
   - Handle singular/plural forms
   - "1 item" vs "2 items"

2. **Date/Number Formatting**
   - Locale-specific formats
   - Currency formatting

3. **Translation Management UI**
   - Admin panel to edit translations
   - Translation export/import

4. **Translation Validation**
   - CI/CD check for missing keys
   - Parameter validation in templates

5. **Hot Reload** (Development)
   - Watch translation files for changes
   - Reload without restart

---

## 16. Decision Records

### ADR-001: Standard Library vs. External i18n Package

**Decision**: Use standard library only (text/template, encoding/json)

**Rationale**:
- Current needs are simple (key-value translation, parameter interpolation)
- Fewer dependencies = simpler, more maintainable
- Performance is acceptable
- Can migrate to external package later if needed

**Consequences**:
- No built-in pluralization (acceptable for now)
- Manual implementation required (more code)
- Full control over behavior

### ADR-002: Load All Translations at Startup

**Decision**: Pre-load all translation files at application startup

**Rationale**:
- Memory footprint is negligible (~15 KB)
- Eliminates I/O during request handling
- Simplifies error handling (fail fast at startup)
- Better performance (no disk access per request)

**Consequences**:
- New translations require application restart
- All languages loaded even if not used
- Simpler code, better performance

### ADR-003: Language Detection Strategy

**Decision**: Priority = Query param > Accept-Language > Default

**Rationale**:
- Query param allows explicit override (useful for testing, sharing links)
- Accept-Language respects browser preferences
- Default ensures system always works

**Consequences**:
- Predictable behavior
- Easy to test
- Follows HTTP standards

---

## 17. Implementation Checklist

### Core Components
- [ ] Implement `internal/infrastructure/i18n/i18n.go` (Service)
- [ ] Implement `internal/handlers/middleware/i18n.go` (Middleware)
- [ ] Create `internal/handlers/dto/i18n_helper.go` (Helper functions)
- [ ] Update `internal/handlers/dto/common.go` (i18n-aware error responses)

### Configuration
- [ ] Add i18n config to `internal/infrastructure/config/config.go`
- [ ] Update `.env.example` with i18n variables

### Integration
- [ ] Update `cmd/api/main.go` to initialize i18n service and middleware
- [ ] Update `internal/handlers/http/user_handler.go` to use i18n

### Testing
- [ ] Write unit tests for i18n.Service
- [ ] Write unit tests for i18n middleware
- [ ] Write integration tests for handlers with i18n
- [ ] Test all three languages (en, pt-BR, es)

### Documentation
- [ ] Update README.md with i18n usage
- [ ] Add inline code documentation
- [ ] Create developer guide for adding new translations

### Validation
- [ ] Test Accept-Language header parsing
- [ ] Test query parameter override
- [ ] Test fallback to default language
- [ ] Test parameter interpolation
- [ ] Test thread safety (concurrent requests)
- [ ] Test missing translation keys

---

## 18. Code Examples

### 18.1 Complete i18n.Service Implementation

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
    T(lang, messageID string, params map[string]interface{}) string
    SupportedLanguages() []string
    DefaultLanguage() string
}

type service struct {
    translations map[string]map[string]string
    mu           sync.RWMutex
    defaultLang  string
}

// NewService creates a new i18n service
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
    lang = lang[:len(lang)-5] // remove ".json"

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
        return msg // Return original if template is invalid
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, params); err != nil {
        return msg // Return original if execution fails
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

---

**Version**: 1.0
**Last Updated**: 05/11/2025
**Author**: Claude Code (Architecture Specification)
**Status**: Ready for Implementation
