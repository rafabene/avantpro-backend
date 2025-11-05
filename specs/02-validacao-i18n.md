# Validação e Internacionalização (i18n)

**Versão**: 2.0
**Data**: 05/11/2025

---

## 1. Visão Geral

Sistema de validação e internacionalização:
- **go-playground/validator** para validação declarativa via tags
- **go-i18n** para tradução de mensagens
- Validações customizadas para regras de domínio
- Mensagens de erro traduzidas automaticamente

---

## 2. Validação com go-playground/validator

### 2.1 Setup Básico

```go
// internal/pkg/validator/validator.go
package validator

import (
    "github.com/go-playground/validator/v10"
    "reflect"
    "strings"
)

var validate *validator.Validate

func init() {
    validate = validator.New()

    // Usar nome do campo JSON nas mensagens de erro
    validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
        if name == "-" {
            return ""
        }
        return name
    })

    // Registrar validações customizadas
    registerCustomValidations()
}

// Validate valida uma struct
func Validate(data interface{}) error {
    return validate.Struct(data)
}

// ValidateVar valida uma variável individual
func ValidateVar(field interface{}, tag string) error {
    return validate.Var(field, tag)
}

// GetValidator retorna instância do validator (para registrar validações customizadas)
func GetValidator() *validator.Validate {
    return validate
}
```

### 2.2 Validações Built-in

```go
// internal/handlers/dto/user_dto.go
package dto

type CreateUserRequest struct {
    // Required
    Email    string `json:"email" binding:"required,email"`
    Name     string `json:"name" binding:"required,min=2,max=100"`
    Password string `json:"password" binding:"required,min=8,max=72"`

    // Optional com defaults
    Role string `json:"role" binding:"omitempty,oneof=admin user guest"`

    // Validações numéricas
    Age int `json:"age" binding:"required,gte=18,lte=120"`

    // Validações de string
    Username string `json:"username" binding:"required,alphanum,min=3,max=30"`

    // URLs
    Website string `json:"website" binding:"omitempty,url"`

    // Datas (RFC3339)
    BirthDate string `json:"birth_date" binding:"required,datetime=2006-01-02"`

    // UUIDs
    ReferralCode string `json:"referral_code" binding:"omitempty,uuid"`
}

type UpdateUserRequest struct {
    // Todos opcionais em updates
    Name     *string `json:"name" binding:"omitempty,min=2,max=100"`
    Age      *int    `json:"age" binding:"omitempty,gte=18,lte=120"`
    Website  *string `json:"website" binding:"omitempty,url"`
}

type AddressRequest struct {
    Street  string `json:"street" binding:"required"`
    Number  string `json:"number" binding:"required"`
    City    string `json:"city" binding:"required"`
    State   string `json:"state" binding:"required,len=2"` // UF com 2 letras
    ZipCode string `json:"zip_code" binding:"required,len=8,numeric"` // CEP 8 dígitos
    Country string `json:"country" binding:"required,iso3166_1_alpha2"` // BR, US, etc
}
```

### 2.3 Validações Customizadas

```go
// internal/pkg/validator/custom.go
package validator

import (
    "regexp"
    "github.com/go-playground/validator/v10"
    "avantpro-backend/internal/domain/valueobjects"
)

func registerCustomValidations() {
    validate.RegisterValidation("cpf", validateCPF)
    validate.RegisterValidation("cnpj", validateCNPJ)
    validate.RegisterValidation("phone_br", validateBrazilianPhone)
    validate.RegisterValidation("strong_password", validateStrongPassword)
}

// validateCPF valida CPF brasileiro
func validateCPF(fl validator.FieldLevel) bool {
    cpf := fl.Field().String()
    _, err := valueobjects.NewCPF(cpf)
    return err == nil
}

// validateCNPJ valida CNPJ brasileiro
func validateCNPJ(fl validator.FieldLevel) bool {
    cnpj := fl.Field().String()
    _, err := valueobjects.NewCNPJ(cnpj)
    return err == nil
}

// validateBrazilianPhone valida telefone brasileiro
func validateBrazilianPhone(fl validator.FieldLevel) bool {
    phone := fl.Field().String()
    // Aceita: (11) 98765-4321 ou 11987654321
    pattern := `^(\(\d{2}\)\s?)?(\d{4,5}-?\d{4})$`
    matched, _ := regexp.MatchString(pattern, phone)
    return matched
}

// validateStrongPassword valida senha forte
func validateStrongPassword(fl validator.FieldLevel) bool {
    password := fl.Field().String()

    // Mínimo 8 caracteres
    if len(password) < 8 {
        return false
    }

    // Pelo menos uma letra maiúscula
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    // Pelo menos uma letra minúscula
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    // Pelo menos um número
    hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
    // Pelo menos um caractere especial
    hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

    return hasUpper && hasLower && hasNumber && hasSpecial
}
```

### 2.4 Validação Cross-Field

```go
// Validação entre campos
type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" binding:"required"`
    NewPassword     string `json:"new_password" binding:"required,strong_password,nefield=CurrentPassword"`
    ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

type DateRangeRequest struct {
    StartDate string `json:"start_date" binding:"required,datetime=2006-01-02"`
    EndDate   string `json:"end_date" binding:"required,datetime=2006-01-02,gtfield=StartDate"`
}

type PriceRangeRequest struct {
    MinPrice float64 `json:"min_price" binding:"required,gte=0"`
    MaxPrice float64 `json:"max_price" binding:"required,gtfield=MinPrice"`
}
```

### 2.5 Validação de Slices e Nested Structs

```go
type CreateOrderRequest struct {
    CustomerID string `json:"customer_id" binding:"required,uuid"`

    // Validar cada item do slice
    Items []OrderItemRequest `json:"items" binding:"required,min=1,dive"`

    // Nested struct
    ShippingAddress AddressRequest `json:"shipping_address" binding:"required"`

    // Optional nested struct
    BillingAddress *AddressRequest `json:"billing_address" binding:"omitempty"`
}

type OrderItemRequest struct {
    ProductID string  `json:"product_id" binding:"required,uuid"`
    Quantity  int     `json:"quantity" binding:"required,gte=1,lte=1000"`
    Price     float64 `json:"price" binding:"required,gt=0"`
}
```

---

## 3. Error Handling e Formatação

### 3.1 Formatador de Erros (RFC 7807)

Erros de validação seguem **RFC 7807 - Problem Details for HTTP APIs**.

```go
// internal/pkg/validator/errors.go
package validator

import (
    "fmt"
    "github.com/go-playground/validator/v10"
)

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Tag     string `json:"tag,omitempty"`
    Value   string `json:"value,omitempty"`
}

// FormatValidationErrors formata erros de validação
func FormatValidationErrors(err error) *ValidationErrors {
    if validationErrors, ok := err.(validator.ValidationErrors); ok {
        errors := make([]ValidationError, 0, len(validationErrors))

        for _, e := range validationErrors {
            errors = append(errors, ValidationError{
                Field:   e.Field(),
                Tag:     e.Tag(),
                Value:   fmt.Sprintf("%v", e.Value()),
                Message: getErrorMessage(e),
            })
        }

        return &ValidationErrors{Errors: errors}
    }

    return nil
}

// getErrorMessage retorna mensagem de erro amigável
func getErrorMessage(e validator.FieldError) string {
    field := e.Field()
    tag := e.Tag()
    param := e.Param()

    switch tag {
    case "required":
        return fmt.Sprintf("%s is required", field)
    case "email":
        return fmt.Sprintf("%s must be a valid email address", field)
    case "min":
        return fmt.Sprintf("%s must be at least %s characters", field, param)
    case "max":
        return fmt.Sprintf("%s must be at most %s characters", field, param)
    case "gte":
        return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
    case "lte":
        return fmt.Sprintf("%s must be less than or equal to %s", field, param)
    case "gt":
        return fmt.Sprintf("%s must be greater than %s", field, param)
    case "lt":
        return fmt.Sprintf("%s must be less than %s", field, param)
    case "len":
        return fmt.Sprintf("%s must be exactly %s characters", field, param)
    case "oneof":
        return fmt.Sprintf("%s must be one of: %s", field, param)
    case "url":
        return fmt.Sprintf("%s must be a valid URL", field)
    case "uuid":
        return fmt.Sprintf("%s must be a valid UUID", field)
    case "alphanum":
        return fmt.Sprintf("%s must contain only letters and numbers", field)
    case "numeric":
        return fmt.Sprintf("%s must contain only numbers", field)
    case "eqfield":
        return fmt.Sprintf("%s must match %s", field, param)
    case "nefield":
        return fmt.Sprintf("%s must be different from %s", field, param)
    case "gtfield":
        return fmt.Sprintf("%s must be greater than %s", field, param)

    // Validações customizadas
    case "cpf":
        return fmt.Sprintf("%s must be a valid CPF", field)
    case "cnpj":
        return fmt.Sprintf("%s must be a valid CNPJ", field)
    case "phone_br":
        return fmt.Sprintf("%s must be a valid Brazilian phone number", field)
    case "strong_password":
        return "Password must contain at least 8 characters, one uppercase, one lowercase, one number, and one special character"

    default:
        return fmt.Sprintf("%s is invalid", field)
    }
}
```

### 3.2 Middleware de Validação

```go
// internal/handlers/middleware/validation.go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
    customValidator "avantpro-backend/internal/pkg/validator"
)

// ValidationErrorHandler processa erros de binding/validação
func ValidationErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        // Se houve erro de validação durante c.ShouldBindJSON
        if len(c.Errors) > 0 {
            err := c.Errors[0].Err

            // Formatar erros de validação
            if validationErrors := customValidator.FormatValidationErrors(err); validationErrors != nil {
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": "validation_error",
                    "details": validationErrors,
                })
                return
            }

            // Outros erros de binding
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "bad_request",
                "message": err.Error(),
            })
        }
    }
}
```

---

## 4. Internacionalização (i18n)

### 4.1 Setup go-i18n

```go
// internal/infrastructure/i18n/i18n.go
package i18n

import (
    "embed"
    "github.com/nicksnyder/go-i18n/v2/i18n"
    "golang.org/x/text/language"
    "encoding/json"
)

//go:embed locales/*.json
var localeFS embed.FS

type I18n struct {
    bundle *i18n.Bundle
}

func NewI18n() *I18n {
    bundle := i18n.NewBundle(language.English)
    bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

    // Carregar arquivos de tradução
    bundle.LoadMessageFileFS(localeFS, "locales/en.json")
    bundle.LoadMessageFileFS(localeFS, "locales/pt-BR.json")
    bundle.LoadMessageFileFS(localeFS, "locales/es.json")

    return &I18n{bundle: bundle}
}

// GetLocalizer retorna localizer para um idioma
func (i *I18n) GetLocalizer(lang string) *i18n.Localizer {
    return i18n.NewLocalizer(i.bundle, lang)
}

// T traduz uma mensagem
func (i *I18n) T(localizer *i18n.Localizer, messageID string, templateData map[string]interface{}) string {
    msg, err := localizer.Localize(&i18n.LocalizeConfig{
        MessageID:    messageID,
        TemplateData: templateData,
    })

    if err != nil {
        return messageID // Fallback para messageID se tradução não existir
    }

    return msg
}
```

### 4.2 Arquivos de Tradução

```json
// internal/infrastructure/i18n/locales/en.json
{
  "welcome": "Welcome, {{.Name}}!",
  "user_created": "User created successfully",
  "password_changed": "Password changed successfully",
  "email_sent": "Email sent to {{.Email}}",

  "validation_required": "{{.Field}} is required",
  "validation_email": "{{.Field}} must be a valid email address",
  "validation_min": "{{.Field}} must be at least {{.Min}} characters",
  "validation_max": "{{.Field}} must be at most {{.Max}} characters",
  "validation_cpf": "{{.Field}} must be a valid CPF",

  "error.user_not_found": "User not found",
  "error.email_already_exists": "Email already in use",
  "error.invalid_credentials": "Invalid email or password",
  "error.unauthorized": "Unauthorized access",
  "error.forbidden": "You don't have permission to access this resource",
  "error.invalid_email": "Invalid email format",
  "error.invalid_cpf": "Invalid CPF",

  "error.validation.title": "Validation Failed",
  "error.validation.detail": "One or more fields failed validation",
  "error.not_found.title": "Resource Not Found",
  "error.not_found.detail": "{{.Resource}} not found",
  "error.conflict.title": "Resource Conflict",
  "error.conflict.email_exists": "The email address is already registered",
  "error.unauthorized.title": "Unauthorized",
  "error.unauthorized.detail": "Authentication is required to access this resource",
  "error.forbidden.title": "Forbidden",
  "error.forbidden.detail": "You don't have permission to access this resource",
  "error.internal.title": "Internal Server Error",
  "error.internal.detail": "An unexpected error occurred while processing your request"
}
```

```json
// internal/infrastructure/i18n/locales/pt-BR.json
{
  "welcome": "Bem-vindo, {{.Name}}!",
  "user_created": "Usuário criado com sucesso",
  "password_changed": "Senha alterada com sucesso",
  "email_sent": "Email enviado para {{.Email}}",

  "validation_required": "{{.Field}} é obrigatório",
  "validation_email": "{{.Field}} deve ser um email válido",
  "validation_min": "{{.Field}} deve ter pelo menos {{.Min}} caracteres",
  "validation_max": "{{.Field}} deve ter no máximo {{.Max}} caracteres",
  "validation_cpf": "{{.Field}} deve ser um CPF válido",

  "error.user_not_found": "Usuário não encontrado",
  "error.email_already_exists": "Email já está em uso",
  "error.invalid_credentials": "Email ou senha inválidos",
  "error.unauthorized": "Acesso não autorizado",
  "error.forbidden": "Você não tem permissão para acessar este recurso",
  "error.invalid_email": "Formato de email inválido",
  "error.invalid_cpf": "CPF inválido",

  "error.validation.title": "Erro de Validação",
  "error.validation.detail": "Um ou mais campos falharam na validação",
  "error.not_found.title": "Recurso Não Encontrado",
  "error.not_found.detail": "{{.Resource}} não encontrado",
  "error.conflict.title": "Conflito de Recurso",
  "error.conflict.email_exists": "O endereço de email já está registrado",
  "error.unauthorized.title": "Não Autorizado",
  "error.unauthorized.detail": "Autenticação é necessária para acessar este recurso",
  "error.forbidden.title": "Proibido",
  "error.forbidden.detail": "Você não tem permissão para acessar este recurso",
  "error.internal.title": "Erro Interno do Servidor",
  "error.internal.detail": "Ocorreu um erro inesperado ao processar sua requisição"
}
```

```json
// internal/infrastructure/i18n/locales/es.json
{
  "welcome": "¡Bienvenido, {{.Name}}!",
  "user_created": "Usuario creado exitosamente",
  "password_changed": "Contraseña cambiada exitosamente",
  "email_sent": "Correo enviado a {{.Email}}",

  "validation_required": "{{.Field}} es obligatorio",
  "validation_email": "{{.Field}} debe ser un correo electrónico válido",
  "validation_min": "{{.Field}} debe tener al menos {{.Min}} caracteres",
  "validation_max": "{{.Field}} debe tener como máximo {{.Max}} caracteres",
  "validation_cpf": "{{.Field}} debe ser un CPF válido",

  "error.user_not_found": "Usuario no encontrado",
  "error.email_already_exists": "El correo electrónico ya está en uso",
  "error.invalid_credentials": "Correo electrónico o contraseña inválidos",
  "error.unauthorized": "Acceso no autorizado",
  "error.forbidden": "No tienes permiso para acceder a este recurso",
  "error.invalid_email": "Formato de correo electrónico inválido",
  "error.invalid_cpf": "CPF inválido",

  "error.validation.title": "Error de Validación",
  "error.validation.detail": "Uno o más campos fallaron en la validación",
  "error.not_found.title": "Recurso No Encontrado",
  "error.not_found.detail": "{{.Resource}} no encontrado",
  "error.conflict.title": "Conflicto de Recurso",
  "error.conflict.email_exists": "La dirección de correo electrónico ya está registrada",
  "error.unauthorized.title": "No Autorizado",
  "error.unauthorized.detail": "Se requiere autenticación para acceder a este recurso",
  "error.forbidden.title": "Prohibido",
  "error.forbidden.detail": "No tienes permiso para acceder a este recurso",
  "error.internal.title": "Error Interno del Servidor",
  "error.internal.detail": "Ocurrió un error inesperado al procesar tu solicitud"
}
```

### 4.3 Middleware de i18n

```go
// internal/handlers/middleware/i18n.go
package middleware

import (
    "strings"
    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/infrastructure/i18n"
)

type I18nMiddleware struct {
    i18n *i18n.I18n
}

func NewI18nMiddleware(i18n *i18n.I18n) *I18nMiddleware {
    return &I18nMiddleware{i18n: i18n}
}

// SetLocalizer detecta idioma e adiciona localizer ao contexto
func (m *I18nMiddleware) SetLocalizer() gin.HandlerFunc {
    return func(c *gin.Context) {
        lang := m.detectLanguage(c)
        localizer := m.i18n.GetLocalizer(lang)

        c.Set("localizer", localizer)
        c.Set("lang", lang)

        c.Next()
    }
}

// detectLanguage detecta idioma preferido do usuário
func (m *I18nMiddleware) detectLanguage(c *gin.Context) string {
    // 1. Query parameter: ?lang=pt-BR
    if lang := c.Query("lang"); lang != "" {
        return lang
    }

    // 2. Header: Accept-Language
    acceptLang := c.GetHeader("Accept-Language")
    if acceptLang != "" {
        // Parsear Accept-Language: pt-BR,pt;q=0.9,en;q=0.8
        parts := strings.Split(acceptLang, ",")
        if len(parts) > 0 {
            lang := strings.Split(parts[0], ";")[0]
            return strings.TrimSpace(lang)
        }
    }

    // 3. Default: English
    return "en"
}
```

### 4.4 Helper para Tradução

```go
// internal/pkg/i18n/helper.go
package i18n

import (
    "github.com/gin-gonic/gin"
    "github.com/nicksnyder/go-i18n/v2/i18n"
)

// T traduz uma mensagem usando localizer do contexto
func T(c *gin.Context, messageID string, templateData ...map[string]interface{}) string {
    localizer, exists := c.Get("localizer")
    if !exists {
        return messageID
    }

    loc := localizer.(*i18n.Localizer)

    var data map[string]interface{}
    if len(templateData) > 0 {
        data = templateData[0]
    }

    msg, err := loc.Localize(&i18n.LocalizeConfig{
        MessageID:    messageID,
        TemplateData: data,
    })

    if err != nil {
        return messageID
    }

    return msg
}

// MustT traduz e entra em panic se falhar (para uso em templates)
func MustT(c *gin.Context, messageID string, templateData ...map[string]interface{}) string {
    result := T(c, messageID, templateData...)
    if result == messageID {
        panic("translation not found: " + messageID)
    }
    return result
}
```

---

## 5. Integração: Validação + i18n

### 5.1 Validador com i18n

```go
// internal/pkg/validator/i18n.go
package validator

import (
    "fmt"
    "github.com/go-playground/validator/v10"
    "github.com/nicksnyder/go-i18n/v2/i18n"
)

// FormatValidationErrorsI18n formata erros com tradução
func FormatValidationErrorsI18n(err error, localizer *i18n.Localizer) *ValidationErrors {
    if validationErrors, ok := err.(validator.ValidationErrors); ok {
        errors := make([]ValidationError, 0, len(validationErrors))

        for _, e := range validationErrors {
            errors = append(errors, ValidationError{
                Field:   e.Field(),
                Tag:     e.Tag(),
                Value:   fmt.Sprintf("%v", e.Value()),
                Message: getErrorMessageI18n(e, localizer),
            })
        }

        return &ValidationErrors{Errors: errors}
    }

    return nil
}

// getErrorMessageI18n retorna mensagem traduzida
func getErrorMessageI18n(e validator.FieldError, localizer *i18n.Localizer) string {
    field := e.Field()
    tag := e.Tag()
    param := e.Param()

    messageID := "validation_" + tag

    // Template data
    data := map[string]interface{}{
        "Field": field,
        "Value": e.Value(),
    }

    // Adicionar parâmetros específicos
    switch tag {
    case "min", "max", "gte", "lte", "gt", "lt", "len":
        data["Min"] = param
        data["Max"] = param
        data["Param"] = param
    case "eqfield", "nefield", "gtfield":
        data["OtherField"] = param
    }

    // Tentar traduzir
    msg, err := localizer.Localize(&i18n.LocalizeConfig{
        MessageID:    messageID,
        TemplateData: data,
    })

    if err != nil {
        // Fallback para mensagem em inglês
        return getErrorMessage(e)
    }

    return msg
}
```

### 5.2 Handler com Validação e i18n

```go
// internal/handlers/http/user_handler.go
package http

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/handlers/dto"
    "avantpro-backend/internal/services"
    customValidator "avantpro-backend/internal/pkg/validator"
    i18nHelper "avantpro-backend/internal/pkg/i18n"
)

func (h *UserHandler) CreateUser(c *gin.Context) {
    var req dto.CreateUserRequest

    // Bind e validação
    if err := c.ShouldBindJSON(&req); err != nil {
        // Pegar localizer do contexto
        localizer, _ := c.Get("localizer")

        // Formatar erros com i18n
        validationErrors := customValidator.FormatValidationErrorsI18n(err, localizer.(*i18n.Localizer))

        // RFC 7807 - Problem Details com i18n
        response := dto.ValidationErrorResponse(c, validationErrors.Errors)
        c.JSON(http.StatusBadRequest, response)
        return
    }

    // Chamar service
    user, err := h.userService.CreateUser(c.Request.Context(), services.CreateUserInput{
        Email: req.Email,
        Name:  req.Name,
    })

    if err != nil {
        // Tratar erros específicos
        var response dto.ErrorResponse

        switch err {
        case services.ErrEmailAlreadyExists:
            // RFC 7807 com i18n - conflito
            response = dto.ConflictErrorResponse(c, "error.conflict.email_exists")
            c.JSON(http.StatusConflict, response)
        default:
            // RFC 7807 com i18n - erro interno
            response = dto.InternalErrorResponse(c)
            c.JSON(http.StatusInternalServerError, response)
        }
        return
    }

    // Sucesso traduzido
    c.JSON(http.StatusCreated, gin.H{
        "message": i18nHelper.T(c, "user_created"),
        "data":    dto.ToUserResponse(user),
    })
}
```

---

## 6. Validação em Domain Layer

### 6.1 Value Objects com Validação

```go
// internal/domain/valueobjects/email.go
package valueobjects

import (
    "errors"
    "regexp"
    "strings"
)

var (
    ErrInvalidEmail = errors.New("invalid email format")
)

type Email struct {
    value string
}

func NewEmail(email string) (Email, error) {
    email = strings.TrimSpace(strings.ToLower(email))

    if !isValidEmail(email) {
        return Email{}, ErrInvalidEmail
    }

    return Email{value: email}, nil
}

func (e Email) String() string {
    return e.value
}

func isValidEmail(email string) bool {
    if len(email) < 3 || len(email) > 254 {
        return false
    }

    pattern := `^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`
    matched, _ := regexp.MatchString(pattern, email)
    return matched
}
```

```go
// internal/domain/valueobjects/cpf.go
package valueobjects

import (
    "errors"
    "regexp"
    "strconv"
)

var (
    ErrInvalidCPF = errors.New("invalid CPF")
)

type CPF struct {
    value string
}

func NewCPF(cpf string) (CPF, error) {
    // Remover caracteres não numéricos
    cpf = regexp.MustCompile(`\D`).ReplaceAllString(cpf, "")

    if !isValidCPF(cpf) {
        return CPF{}, ErrInvalidCPF
    }

    return CPF{value: cpf}, nil
}

func (c CPF) String() string {
    return c.value
}

func (c CPF) Formatted() string {
    // 123.456.789-10
    return c.value[:3] + "." + c.value[3:6] + "." + c.value[6:9] + "-" + c.value[9:]
}

func isValidCPF(cpf string) bool {
    if len(cpf) != 11 {
        return false
    }

    // Verificar se todos os dígitos são iguais
    allEqual := true
    for i := 1; i < len(cpf); i++ {
        if cpf[i] != cpf[0] {
            allEqual = false
            break
        }
    }
    if allEqual {
        return false
    }

    // Calcular dígitos verificadores
    return validateCPFDigits(cpf)
}

func validateCPFDigits(cpf string) bool {
    // Primeiro dígito verificador
    sum := 0
    for i := 0; i < 9; i++ {
        num, _ := strconv.Atoi(string(cpf[i]))
        sum += num * (10 - i)
    }
    remainder := sum % 11
    digit1 := 0
    if remainder >= 2 {
        digit1 = 11 - remainder
    }

    if digit1 != int(cpf[9]-'0') {
        return false
    }

    // Segundo dígito verificador
    sum = 0
    for i := 0; i < 10; i++ {
        num, _ := strconv.Atoi(string(cpf[i]))
        sum += num * (11 - i)
    }
    remainder = sum % 11
    digit2 := 0
    if remainder >= 2 {
        digit2 = 11 - remainder
    }

    return digit2 == int(cpf[10]-'0')
}
```

### 6.2 Entity Validation

```go
// internal/domain/entities/user.go
package entities

import "errors"

var (
    ErrInvalidUserData = errors.New("invalid user data")
)

// Validate valida regras de negócio
func (u *User) Validate() error {
    if u.Email.String() == "" {
        return errors.New("email is required")
    }

    if u.Name == "" {
        return errors.New("name is required")
    }

    if len(u.Name) < 2 {
        return errors.New("name must be at least 2 characters")
    }

    if u.Role != RoleAdmin && u.Role != RoleUser && u.Role != RoleGuest {
        return errors.New("invalid role")
    }

    return nil
}
```

---

## 7. Exemplos de Uso

### 7.1 Setup Completo

```go
// cmd/api/main.go
package main

import (
    "github.com/gin-gonic/gin"
    "avantpro-backend/internal/handlers/middleware"
    "avantpro-backend/internal/infrastructure/i18n"
)

func main() {
    router := gin.Default()

    // Setup i18n
    i18nService := i18n.NewI18n()
    i18nMiddleware := middleware.NewI18nMiddleware(i18nService)

    // Middlewares globais
    router.Use(i18nMiddleware.SetLocalizer())
    router.Use(middleware.ValidationErrorHandler())

    // Routes
    setupRoutes(router)

    router.Run(":8080")
}
```

### 7.2 Exemplo Completo de Validação (RFC 7807)

```http
// Request
POST /api/v1/users
Accept-Language: pt-BR
Content-Type: application/json

{
  "email": "invalid-email",
  "name": "J",
  "password": "weak",
  "cpf": "123.456.789-00"
}

// Response - RFC 7807 Problem Details
HTTP/1.1 400 Bad Request
Content-Type: application/problem+json

{
  "type": "https://api.avantpro.com/problems/validation-error",
  "title": "Erro de Validação",
  "status": 400,
  "detail": "Um ou mais campos falharam na validação",
  "instance": "/api/v1/users",
  "errors": [
    {
      "field": "email",
      "message": "email deve ser um email válido",
      "tag": "email"
    },
    {
      "field": "name",
      "message": "name deve ter pelo menos 2 caracteres",
      "tag": "min"
    },
    {
      "field": "password",
      "message": "Senha deve conter pelo menos 8 caracteres, uma letra maiúscula, uma minúscula, um número e um caractere especial",
      "tag": "strong_password"
    },
    {
      "field": "cpf",
      "message": "cpf deve ser um CPF válido",
      "tag": "cpf"
    }
  ]
}
```

**Nota**: O `Content-Type` deve ser `application/problem+json` conforme RFC 7807.

---

## 8. Best Practices

### 8.1 Checklist

- ✅ Usar tags de validação em DTOs
- ✅ Validações customizadas em `internal/pkg/validator/custom.go`
- ✅ Value Objects para validações de domínio
- ✅ Mensagens de erro traduzidas
- ✅ Accept-Language header para detectar idioma
- ✅ Fallback para inglês se tradução não existir
- ✅ Validação no handler (formato) + service (negócio)
- ✅ **Erros seguem RFC 7807** (type, title, status, detail, instance)
- ✅ Content-Type: application/problem+json para erros
- ✅ Não expor stack traces em produção
- ✅ Logging de erros de validação para métricas

### 8.2 Quando Validar Onde

| Camada | Tipo de Validação | Exemplo |
|--------|-------------------|---------|
| **Handler/DTO** | Formato, tipo, required | Email válido, número positivo |
| **Service** | Regras de negócio simples | Email já existe, saldo suficiente |
| **Domain** | Regras de domínio complexas | CPF válido, pedido pode ser cancelado |
| **Value Object** | Invariantes do valor | Email, CPF, Money sempre válidos |

---

**Versão**: 2.0
**Última Atualização**: 05/11/2025
