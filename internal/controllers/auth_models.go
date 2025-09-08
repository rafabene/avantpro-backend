package controllers

import (
	"time"

	"github.com/google/uuid"
)

// UserResponse representa o corpo da resposta para operações de usuário
// @Description Resposta do usuário
type UserResponse struct {
	ID                         uuid.UUID        `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username                   string           `json:"username" example:"user@example.com"`
	Name                       string           `json:"name" example:"John Doe"`
	LastSelectedOrganizationID *uuid.UUID       `json:"last_selected_organization_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	Profile                    *ProfileResponse `json:"profile,omitempty"`
	CreatedAt                  time.Time        `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt                  time.Time        `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// ProfileResponse representa o corpo da resposta para operações de perfil
// @Description Resposta do perfil
type ProfileResponse struct {
	ID        uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Street    string    `json:"street" example:"123 Main Street"`
	City      string    `json:"city" example:"São Paulo"`
	District  string    `json:"district" example:"Centro"`
	ZipCode   string    `json:"zip_code" example:"01234567"`
	Phone     string    `json:"phone" example:"11987654321"`
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// LoginRequest representa o corpo da requisição para login do usuário
// @Description Requisição de login do usuário
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"password123"`
}

// RegisterRequest representa o corpo da requisição para registro do usuário
// @Description Requisição de registro do usuário
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Name     string `json:"name" validate:"required,min=2,max=100" example:"John Doe"`
	Password string `json:"password" validate:"required" example:"SecurePass123!"`
}

// LoginResponse representa o corpo da resposta para login bem-sucedido
// @Description Resposta de login com token e informações do usuário
type LoginResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  UserResponse `json:"user"`
}

// PasswordResetRequest representa o corpo da requisição para redefinição de senha
// @Description Requisição de redefinição de senha
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

// PasswordResetConfirmRequest representa o corpo da requisição para confirmação de redefinição de senha
// @Description Requisição de confirmação de redefinição de senha
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" validate:"required" example:"reset-token-123"`
	NewPassword string `json:"new_password" validate:"required" example:"NewSecurePass123!"`
}

// MessageResponse representa uma resposta de mensagem simples
// @Description Resposta de mensagem simples
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// UpdateLastSelectedOrganizationRequest representa o corpo da requisição para atualizar a última organização selecionada
// @Description Requisição de atualização da última organização selecionada
type UpdateLastSelectedOrganizationRequest struct {
	OrganizationID *uuid.UUID `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440002"`
}

// Tipos de conversão para services (usados internamente nos controllers)
type ServiceLoginRequest struct {
	Email    string
	Password string
}

type ServiceRegisterRequest struct {
	Email    string
	Name     string
	Password string
}

type ServicePasswordResetRequest struct {
	Email string
}

type ServicePasswordResetConfirmRequest struct {
	Token       string
	NewPassword string
}

type ServiceUpdateLastSelectedOrganizationRequest struct {
	OrganizationID *uuid.UUID
}
