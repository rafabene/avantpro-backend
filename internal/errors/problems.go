package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/moogar0880/problems"
)

// ProblemDetail representa um detalhe de problema RFC 7807 para documentação Swagger
// @Description Resposta de erro seguindo RFC 7807 Problem Details para APIs HTTP
type ProblemDetail struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/internal"`
	Title    string `json:"title" example:"Erro Interno do Servidor"`
	Status   int    `json:"status" example:"500"`
	Detail   string `json:"detail" example:"Ocorreu um erro inesperado. Tente novamente mais tarde."`
	Instance string `json:"instance,omitempty" example:"/api/v1/notifications"`
}

// BadRequestProblem representa um erro 400 Bad Request
// @Description Resposta de erro de requisição inválida
type BadRequestProblem struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/bad-request"`
	Title    string `json:"title" example:"Requisição Inválida"`
	Status   int    `json:"status" example:"400"`
	Detail   string `json:"detail" example:"Formato de requisição ou parâmetros inválidos"`
	Instance string `json:"instance,omitempty" example:"/api/v1/notifications"`
}

// UnauthorizedProblem representa um erro 401 Unauthorized
// @Description Resposta de erro não autorizado
type UnauthorizedProblem struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/unauthorized"`
	Title    string `json:"title" example:"Não Autorizado"`
	Status   int    `json:"status" example:"401"`
	Detail   string `json:"detail" example:"Autenticação necessária ou token inválido"`
	Instance string `json:"instance,omitempty" example:"/api/v1/notifications"`
}

// ForbiddenProblem representa um erro 403 Forbidden
// @Description Resposta de erro proibido
type ForbiddenProblem struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/forbidden"`
	Title    string `json:"title" example:"Proibido"`
	Status   int    `json:"status" example:"403"`
	Detail   string `json:"detail" example:"Acesso negado para este recurso"`
	Instance string `json:"instance,omitempty" example:"/api/v1/organizations"`
}

// NotFoundProblem representa um erro 404 Not Found
// @Description Resposta de erro não encontrado
type NotFoundProblem struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/not-found"`
	Title    string `json:"title" example:"Recurso Não Encontrado"`
	Status   int    `json:"status" example:"404"`
	Detail   string `json:"detail" example:"O recurso solicitado não foi encontrado"`
	Instance string `json:"instance,omitempty" example:"/api/v1/notifications/123"`
}

// ConflictProblem representa um erro 409 Conflict
// @Description Resposta de erro de conflito
type ConflictProblem struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/conflict"`
	Title    string `json:"title" example:"Conflito de Recurso"`
	Status   int    `json:"status" example:"409"`
	Detail   string `json:"detail" example:"Recurso já existe ou entra em conflito com o estado atual"`
	Instance string `json:"instance,omitempty" example:"/api/v1/users"`
}

// GoneProblem representa um erro 410 Gone
// @Description Resposta de erro de recurso indisponível
type GoneProblem struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/gone"`
	Title    string `json:"title" example:"Recurso Indisponível"`
	Status   int    `json:"status" example:"410"`
	Detail   string `json:"detail" example:"Recurso não está mais disponível (ex: convite expirado)"`
	Instance string `json:"instance,omitempty" example:"/api/v1/invites/token/abc123"`
}

// InternalServerProblem representa um erro 500 Internal Server Error
// @Description Resposta de erro interno do servidor
type InternalServerProblem struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/internal"`
	Title    string `json:"title" example:"Erro Interno do Servidor"`
	Status   int    `json:"status" example:"500"`
	Detail   string `json:"detail" example:"Ocorreu um erro inesperado. Tente novamente mais tarde."`
	Instance string `json:"instance,omitempty" example:"/api/v1/notifications"`
}

// Tipos de erro comuns para a API
const (
	// Type URIs for different error categories
	ValidationErrorType   = "https://avantpro-backend.com/errors/validation"
	NotFoundErrorType     = "https://avantpro-backend.com/errors/not-found"
	ConflictErrorType     = "https://avantpro-backend.com/errors/conflict"
	InternalErrorType     = "https://avantpro-backend.com/errors/internal"
	BadRequestErrorType   = "https://avantpro-backend.com/errors/bad-request"
	UnauthorizedErrorType = "https://avantpro-backend.com/errors/unauthorized"
	ForbiddenErrorType    = "https://avantpro-backend.com/errors/forbidden"
	GoneErrorType         = "https://avantpro-backend.com/errors/gone"
)

// Mensagens de erro comuns
const (
	ErrUsernameAlreadyExists = "nome de usuário já existe"
)

// ValidationError creates a validation error problem
func ValidationError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusBadRequest, detail)
	prob.Type = ValidationErrorType
	prob.Title = "Erro de Validação"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// NotFoundError creates a not found error problem
func NotFoundError(resource string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusNotFound, fmt.Sprintf("%s not found", resource))
	prob.Type = NotFoundErrorType
	prob.Title = "Recurso Não Encontrado"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// ConflictError creates a conflict error problem (e.g., duplicate username)
func ConflictError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusConflict, detail)
	prob.Type = ConflictErrorType
	prob.Title = "Conflito de Recurso"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// InternalError creates an internal server error problem
func InternalError(instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusInternalServerError, "Ocorreu um erro inesperado. Tente novamente mais tarde.")
	prob.Type = InternalErrorType
	prob.Title = "Erro Interno do Servidor"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// BadRequestError creates a bad request error problem
func BadRequestError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusBadRequest, detail)
	prob.Type = BadRequestErrorType
	prob.Title = "Requisição Inválida"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// UnauthorizedError creates an unauthorized error problem
func UnauthorizedError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusUnauthorized, detail)
	prob.Type = UnauthorizedErrorType
	prob.Title = "Não Autorizado"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// ForbiddenError creates a forbidden error problem
func ForbiddenError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusForbidden, detail)
	prob.Type = ForbiddenErrorType
	prob.Title = "Proibido"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// GoneError creates a gone error problem (e.g., expired invitations)
func GoneError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusGone, detail)
	prob.Type = GoneErrorType
	prob.Title = "Recurso Indisponível"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// RespondWithProblem sends a problem details response using Gin
func RespondWithProblem(c *gin.Context, prob *problems.Problem) {
	// Definir Content-Type como application/problem+json conforme RFC 7807
	c.Header("Content-Type", "application/problem+json")
	c.JSON(prob.Status, prob)
}

// Auxiliar para extrair caminho da instância do contexto Gin
func GetInstance(c *gin.Context) string {
	return c.Request.URL.Path
}

// FormatValidationError converts validator.ValidationErrors to user-friendly messages
func FormatValidationError(err error) error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var messages []string
	for _, validationErr := range validationErrors {
		field := validationErr.Field()
		tag := validationErr.Tag()
		param := validationErr.Param()

		// Converter nome do campo para nome amigável
		friendlyName := strings.ToLower(field)

		message := formatFieldError(friendlyName, tag, param)
		if message != "" {
			messages = append(messages, message)
		}
	}

	if len(messages) > 0 {
		return errors.New(strings.Join(messages, ", "))
	}

	return err
}

// formatFieldError formats individual field validation errors
func formatFieldError(fieldName, tag, param string) string {
	switch tag {
	case "required":
		return fmt.Sprintf("%s é obrigatório", fieldName)
	case "min":
		return fmt.Sprintf("%s deve ter pelo menos %s caracteres", fieldName, param)
	case "max":
		return fmt.Sprintf("%s deve ter no máximo %s caracteres", fieldName, param)
	case "email":
		return fmt.Sprintf("%s deve ser um endereço de email válido", fieldName)
	case "len":
		return fmt.Sprintf("%s deve ter exatamente %s caracteres", fieldName, param)
	case "gt":
		return fmt.Sprintf("%s deve ser maior que %s", fieldName, param)
	case "gte":
		return fmt.Sprintf("%s deve ser maior ou igual a %s", fieldName, param)
	case "lt":
		return fmt.Sprintf("%s deve ser menor que %s", fieldName, param)
	case "lte":
		return fmt.Sprintf("%s deve ser menor ou igual a %s", fieldName, param)
	case "alphanum":
		return fmt.Sprintf("%s deve conter apenas caracteres alfanuméricos", fieldName)
	case "alpha":
		return fmt.Sprintf("%s deve conter apenas caracteres alfabéticos", fieldName)
	case "numeric":
		return fmt.Sprintf("%s deve conter apenas caracteres numéricos", fieldName)
	case "url":
		return fmt.Sprintf("%s deve ser uma URL válida", fieldName)
	case "uuid":
		return fmt.Sprintf("%s deve ser um UUID válido", fieldName)
	default:
		return fmt.Sprintf("%s é inválido", fieldName)
	}
}

// Manipuladores de conveniência para controllers Gin

// HandleValidationError handles validation errors
func HandleValidationError(c *gin.Context, detail string) {
	prob := ValidationError(detail, GetInstance(c))
	RespondWithProblem(c, prob)
}

// HandleNotFoundError handles not found errors
func HandleNotFoundError(c *gin.Context, detail string) {
	prob := NotFoundError(detail, GetInstance(c))
	RespondWithProblem(c, prob)
}

// HandleConflictError handles conflict errors
func HandleConflictError(c *gin.Context, detail string) {
	prob := ConflictError(detail, GetInstance(c))
	RespondWithProblem(c, prob)
}

// HandleInternalError handles internal server errors
func HandleInternalError(c *gin.Context, detail string, err error) {
	// Registrar o erro real para debug (não expor ao cliente)
	if err != nil {
		// TODO: Add proper logging
		fmt.Printf("Internal error: %v\n", err)
	}
	prob := InternalError(GetInstance(c))
	RespondWithProblem(c, prob)
}

// HandleForbiddenError handles forbidden errors
func HandleForbiddenError(c *gin.Context, detail string) {
	prob := ForbiddenError(detail, GetInstance(c))
	RespondWithProblem(c, prob)
}

// HandleGoneError handles gone errors
func HandleGoneError(c *gin.Context, detail string) {
	prob := GoneError(detail, GetInstance(c))
	RespondWithProblem(c, prob)
}

// HandleUnauthorizedError handles unauthorized errors
func HandleUnauthorizedError(c *gin.Context, detail string) {
	prob := UnauthorizedError(detail, GetInstance(c))
	RespondWithProblem(c, prob)
}
