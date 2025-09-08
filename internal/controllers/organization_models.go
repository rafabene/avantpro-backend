package controllers

import (
	"time"

	"github.com/google/uuid"
)

// OrganizationCreateRequest representa o corpo da requisição para criar uma organização
// @Description Requisição de criação de organização
type OrganizationCreateRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100" example:"My Company"`
	Description string `json:"description" validate:"max=500" example:"A great company"`
}

// OrganizationUpdateRequest representa o corpo da requisição para atualizar uma organização
// @Description Requisição de atualização de organização
type OrganizationUpdateRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=100" example:"My Company"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500" example:"A great company"`
}

// OrganizationInviteRequest representa o corpo da requisição para convidar um usuário para uma organização
// @Description Requisição de convite da organização
type OrganizationInviteRequest struct {
	Email string           `json:"email" validate:"required,email" example:"user@example.com"`
	Role  OrganizationRole `json:"role" validate:"required,oneof=admin user" example:"user"`
}

// OrganizationMemberUpdateRequest representa o corpo da requisição para atualizar o papel de um membro
// @Description Requisição de atualização de membro da organização
type OrganizationMemberUpdateRequest struct {
	Role OrganizationRole `json:"role" validate:"required,oneof=admin user" example:"user"`
}

// OrganizationResponse representa o corpo da resposta para operações de organização
// @Description Resposta da organização
type OrganizationResponse struct {
	ID          uuid.UUID     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string        `json:"name" example:"My Company"`
	Description string        `json:"description" example:"A great company"`
	CreatedBy   uuid.UUID     `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440001"`
	Creator     *UserResponse `json:"creator,omitempty"`
	MemberCount int           `json:"member_count" example:"5"`
	InviteCount int           `json:"invite_count" example:"2"`
	CreatedAt   time.Time     `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time     `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// OrganizationMemberResponse representa o corpo da resposta para operações de membros da organização
// @Description Resposta do membro da organização
type OrganizationMemberResponse struct {
	ID             uuid.UUID             `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationID uuid.UUID             `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	UserID         uuid.UUID             `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Role           OrganizationRole      `json:"role" example:"user"`
	User           *UserResponse         `json:"user,omitempty"`
	Organization   *OrganizationResponse `json:"organization,omitempty"`
	JoinedAt       time.Time             `json:"joined_at" example:"2023-01-01T12:00:00Z"`
	CreatedAt      time.Time             `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt      time.Time             `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// OrganizationInviteResponse representa o corpo da resposta para operações de convites da organização
// @Description Resposta do convite da organização
type OrganizationInviteResponse struct {
	ID             uuid.UUID             `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationID uuid.UUID             `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Email          string                `json:"email" example:"user@example.com"`
	Role           OrganizationRole      `json:"role" example:"user"`
	InvitedBy      uuid.UUID             `json:"invited_by" example:"550e8400-e29b-41d4-a716-446655440002"`
	Status         InviteStatus          `json:"status" example:"pending"`
	ExpiresAt      time.Time             `json:"expires_at" example:"2023-12-31T23:59:59Z"`
	AcceptedAt     *time.Time            `json:"accepted_at,omitempty" example:"2023-01-01T12:00:00Z"`
	Organization   *OrganizationResponse `json:"organization,omitempty"`
	Inviter        *UserResponse         `json:"inviter,omitempty"`
	CreatedAt      time.Time             `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt      time.Time             `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// OrganizationListResponse representa o corpo da resposta para listar organizações
// @Description Resposta da lista de organizações com paginação
type OrganizationListResponse struct {
	Data  []OrganizationResponse `json:"data"`
	Total int64                  `json:"total" example:"100"`
	Limit int                    `json:"limit" example:"50"`
	Page  int                    `json:"page" example:"1"`
}

// OrganizationMemberListResponse representa o corpo da resposta para listar membros da organização
// @Description Resposta da lista de membros da organização com paginação
type OrganizationMemberListResponse struct {
	Data  []OrganizationMemberResponse `json:"data"`
	Total int64                        `json:"total" example:"100"`
	Limit int                          `json:"limit" example:"50"`
	Page  int                          `json:"page" example:"1"`
}

// OrganizationInviteListResponse representa o corpo da resposta para listar convites da organização
// @Description Resposta da lista de convites da organização com paginação
type OrganizationInviteListResponse struct {
	Data  []OrganizationInviteResponse `json:"data"`
	Total int64                        `json:"total" example:"100"`
	Limit int                          `json:"limit" example:"50"`
	Page  int                          `json:"page" example:"1"`
}

// Tipos de conversão para services (usados internamente nos controllers)
type ServiceOrganizationCreateRequest struct {
	Name        string
	Description string
}

type ServiceOrganizationUpdateRequest struct {
	Name        *string
	Description *string
}

type ServiceOrganizationInviteRequest struct {
	Email string
	Role  OrganizationRole
}

type ServiceOrganizationMemberUpdateRequest struct {
	Role OrganizationRole
}
