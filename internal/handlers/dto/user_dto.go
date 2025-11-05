package dto

import (
	"time"

	"github.com/rafabene/avantpro-backend/internal/domain/entities"
)

// CreateUserRequest representa a requisição para criar um usuário
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// UpdateUserRequest representa a requisição para atualizar um usuário
type UpdateUserRequest struct {
	Name *string `json:"name" binding:"omitempty,min=2,max=100"`
}

// UserResponse representa a resposta de um usuário
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ToUserResponse converte uma entidade User para UserResponse
func ToUserResponse(user *entities.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email.String(),
		Name:      user.Name,
		Role:      string(user.Role),
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
	}
}

// ToUserResponses converte uma lista de entidades User para UserResponse
func ToUserResponses(users []*entities.User) []UserResponse {
	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = ToUserResponse(user)
	}
	return responses
}
