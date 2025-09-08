package services

import "errors"

// Erros customizados para autenticação - evitam comparação frágil de strings
var (
	// Erros de autenticação
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrAccountLocked      = errors.New("conta bloqueada")
	ErrUserAlreadyExists  = errors.New("usuário já existe")

	// Erros de reset de senha
	ErrTokenInvalidOrExpired = errors.New("token inválido ou expirado")
	ErrTokenNotFound         = errors.New("token não encontrado")
	ErrTokenUsed             = errors.New("token já foi utilizado")
	ErrTokenExpired          = errors.New("token expirado")

	// Erros de validação
	ErrInvalidPassword = errors.New("senha inválida")
	ErrUserNotFound    = errors.New("usuário não encontrado")

	// Erros internos
	ErrDatabaseError = errors.New("erro de banco de dados")
)
