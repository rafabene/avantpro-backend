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

// ProblemDetail represents an RFC 7807 problem detail for Swagger documentation
// @Description Error response following RFC 7807 Problem Details for HTTP APIs
type ProblemDetail struct {
	Type     string `json:"type" example:"https://avantpro-backend.com/errors/validation"`
	Title    string `json:"title" example:"Validation Error"`
	Status   int    `json:"status" example:"400"`
	Detail   string `json:"detail" example:"Invalid input data"`
	Instance string `json:"instance,omitempty" example:"/api/v1/users"`
}

// Common error types for the API
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

// Common error messages
const (
	ErrUsernameAlreadyExists = "username already exists"
	ErrUserNotFound          = "user not found"
	ErrInvalidCredentials    = "invalid credentials"
	ErrWeakPassword          = "password must be at least 6 characters long"
)

// ValidationError creates a validation error problem
func ValidationError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusBadRequest, detail)
	prob.Type = ValidationErrorType
	prob.Title = "Validation Error"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// NotFoundError creates a not found error problem
func NotFoundError(resource string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusNotFound, fmt.Sprintf("%s not found", resource))
	prob.Type = NotFoundErrorType
	prob.Title = "Resource Not Found"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// ConflictError creates a conflict error problem (e.g., duplicate username)
func ConflictError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusConflict, detail)
	prob.Type = ConflictErrorType
	prob.Title = "Resource Conflict"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// InternalError creates an internal server error problem
func InternalError(instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
	prob.Type = InternalErrorType
	prob.Title = "Internal Server Error"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// BadRequestError creates a bad request error problem
func BadRequestError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusBadRequest, detail)
	prob.Type = BadRequestErrorType
	prob.Title = "Bad Request"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// UnauthorizedError creates an unauthorized error problem
func UnauthorizedError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusUnauthorized, detail)
	prob.Type = UnauthorizedErrorType
	prob.Title = "Unauthorized"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// ForbiddenError creates a forbidden error problem
func ForbiddenError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusForbidden, detail)
	prob.Type = ForbiddenErrorType
	prob.Title = "Forbidden"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// GoneError creates a gone error problem (e.g., expired invitations)
func GoneError(detail string, instance ...string) *problems.Problem {
	prob := problems.NewDetailedProblem(http.StatusGone, detail)
	prob.Type = GoneErrorType
	prob.Title = "Resource Gone"
	if len(instance) > 0 {
		prob.Instance = instance[0]
	}
	return prob
}

// RespondWithProblem sends a problem details response using Gin
func RespondWithProblem(c *gin.Context, prob *problems.Problem) {
	// Set the Content-Type to application/problem+json as per RFC 7807
	c.Header("Content-Type", "application/problem+json")
	c.JSON(prob.Status, prob)
}

// Helper to extract instance path from Gin context
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

		// Convert field name to friendly name
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
		return fmt.Sprintf("%s is required", fieldName)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", fieldName, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", fieldName, param)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fieldName)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", fieldName, param)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", fieldName, param)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", fieldName, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", fieldName, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", fieldName, param)
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", fieldName)
	case "alpha":
		return fmt.Sprintf("%s must contain only alphabetic characters", fieldName)
	case "numeric":
		return fmt.Sprintf("%s must contain only numeric characters", fieldName)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fieldName)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", fieldName)
	default:
		return fmt.Sprintf("%s is invalid", fieldName)
	}
}

// Convenience handlers for Gin controllers

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
	// Log the actual error for debugging (don't expose to client)
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
