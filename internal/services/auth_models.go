package services

import (
	"time"

	"github.com/google/uuid"
)

// UserResponse representa o corpo da resposta para operações de usuário
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
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"password123"`
}

// RegisterRequest representa o corpo da requisição para registro do usuário
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Name     string `json:"name" validate:"required,min=2,max=100" example:"John Doe"`
	Password string `json:"password" validate:"required,min=8,containsany=ABCDEFGHIJKLMNOPQRSTUVWXYZ,containsany=abcdefghijklmnopqrstuvwxyz,containsany=0123456789,containsany=!@#$%^&*()_+-=[]{}|;:,.<>?" example:"SecurePass123!"`
}

// LoginResponse representa o corpo da resposta para login bem-sucedido
type LoginResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  UserResponse `json:"user"`
}

// PasswordResetRequest representa o corpo da requisição para redefinição de senha
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

// PasswordResetConfirmRequest representa o corpo da requisição para confirmação de redefinição de senha
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" validate:"required" example:"reset-token-123"`
	NewPassword string `json:"new_password" validate:"required,min=8,containsany=ABCDEFGHIJKLMNOPQRSTUVWXYZ,containsany=abcdefghijklmnopqrstuvwxyz,containsany=0123456789,containsany=!@#$%^&*()_+-=[]{}|;:,.<>?" example:"NewSecurePass123!"`
}

// MessageResponse representa uma resposta de mensagem simples
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// UpdateLastSelectedOrganizationRequest representa o corpo da requisição para atualizar a última organização selecionada
type UpdateLastSelectedOrganizationRequest struct {
	OrganizationID *uuid.UUID `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440002"`
}
