package errors

import "errors"

// Business errors
// Nota: Estes são códigos de erro (message IDs para i18n).
// As traduções devem estar em internal/infrastructure/i18n/locales/*.json
var (
	ErrUserNotFound       = errors.New("error.user_not_found")
	ErrEmailAlreadyExists = errors.New("error.email_already_exists")
	ErrInvalidCredentials = errors.New("error.invalid_credentials")
	ErrUnauthorized       = errors.New("error.unauthorized")
	ErrForbidden          = errors.New("error.forbidden")
)

// Domain errors
// Nota: Estes são códigos de erro (message IDs para i18n).
// As traduções devem estar em internal/infrastructure/i18n/locales/*.json
var (
	ErrInvalidEmail = errors.New("error.invalid_email")
	ErrInvalidCPF   = errors.New("error.invalid_cpf")
)

// ProblemType define tipos de problemas (URIs RFC 7807)
// Nota: O domínio base virá de configuração (API_BASE_URL)
//
//nolint:misspell
const (
	ProblemTypeValidation   = "/problems/validation-error"
	ProblemTypeNotFound     = "/problems/not-found"
	ProblemTypeConflict     = "/problems/conflict"
	ProblemTypeUnauthorized = "/problems/unauthorized"
	ProblemTypeForbidden    = "/problems/forbidden"
	ProblemTypeInternal     = "/problems/internal-error"
	ProblemTypeBadRequest   = "/problems/bad-request"
)

// DomainError representa um erro de domínio com contexto adicional
type DomainError struct {
	Type    string
	Title   string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Err
}
