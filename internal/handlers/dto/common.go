package dto

import (
	"github.com/gin-gonic/gin"
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

// ValidationError representa um erro de validação de campo
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
	Value   string `json:"value,omitempty"`
}

// NewErrorResponse cria uma nova resposta de erro RFC 7807
func NewErrorResponse(c *gin.Context, problemType, title string, status int, detail string) ErrorResponse {
	// Pegar base URL da configuração
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return ErrorResponse{
		Type:     baseURL + problemType,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: c.Request.URL.Path,
	}
}

// NewErrorResponseI18n cria uma resposta de erro usando i18n
func NewErrorResponseI18n(c *gin.Context, problemType, titleKey, detailKey string, status int, params ...map[string]interface{}) ErrorResponse {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	title := T(c, titleKey, params...)
	detail := T(c, detailKey, params...)

	return ErrorResponse{
		Type:     baseURL + problemType,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: c.Request.URL.Path,
	}
}

// Helper functions para respostas de erro comuns com i18n

// ValidationErrorResponseI18n cria uma resposta de erro de validação
func ValidationErrorResponseI18n(c *gin.Context, validationErrors []ValidationError) ErrorResponse {
	response := NewErrorResponseI18n(
		c,
		"/problems/validation-error",
		"error.validation.title",
		"error.validation.detail",
		400,
	)
	response.Errors = validationErrors
	return response
}

// NotFoundErrorResponseI18n cria uma resposta de erro 404
func NotFoundErrorResponseI18n(c *gin.Context, resource string) ErrorResponse {
	return NewErrorResponseI18n(
		c,
		"/problems/not-found",
		"error.not_found.title",
		"error.not_found.detail",
		404,
		map[string]interface{}{"Resource": resource},
	)
}

// ConflictErrorResponseI18n cria uma resposta de erro 409
func ConflictErrorResponseI18n(c *gin.Context, detailKey string, params ...map[string]interface{}) ErrorResponse {
	return NewErrorResponseI18n(
		c,
		"/problems/conflict",
		"error.conflict.title",
		detailKey,
		409,
		params...,
	)
}

// UnauthorizedErrorResponseI18n cria uma resposta de erro 401
func UnauthorizedErrorResponseI18n(c *gin.Context) ErrorResponse {
	return NewErrorResponseI18n(
		c,
		"/problems/unauthorized",
		"error.unauthorized.title",
		"error.unauthorized.detail",
		401,
	)
}

// ForbiddenErrorResponseI18n cria uma resposta de erro 403
func ForbiddenErrorResponseI18n(c *gin.Context) ErrorResponse {
	return NewErrorResponseI18n(
		c,
		"/problems/forbidden",
		"error.forbidden.title",
		"error.forbidden.detail",
		403,
	)
}

// InternalErrorResponseI18n cria uma resposta de erro 500
func InternalErrorResponseI18n(c *gin.Context) ErrorResponse {
	return NewErrorResponseI18n(
		c,
		"/problems/internal-error",
		"error.internal.title",
		"error.internal.detail",
		500,
	)
}
