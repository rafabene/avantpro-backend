package controllers

import (
	"time"

	"github.com/google/uuid"
)

// NotificationResponse representa o corpo da resposta para operações de notificação
// @Description Resposta da notificação
type NotificationResponse struct {
	ID             uuid.UUID        `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title          string           `json:"title" example:"Novo membro na organização"`
	Message        string           `json:"message" example:"João Silva foi adicionado à organização TechCorp"`
	Type           NotificationType `json:"type" example:"info"`
	Read           bool             `json:"read" example:"false"`
	ReadAt         *time.Time       `json:"read_at,omitempty" example:"2023-01-01T12:00:00Z"`
	Data           string           `json:"data,omitempty" example:"{\"action\":\"member_joined\",\"member_id\":\"123\"}"`
	OrganizationID uuid.UUID        `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	CreatedAt      time.Time        `json:"created_at" example:"2023-01-01T12:00:00Z"`
}

// CreateNotificationRequest representa o corpo da requisição para criar uma notificação
// @Description Requisição de criação de notificação
type CreateNotificationRequest struct {
	UserID         uuid.UUID        `json:"user_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	OrganizationID uuid.UUID        `json:"organization_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440002"`
	Title          string           `json:"title" validate:"required,min=1,max=200" example:"Novo membro na organização"`
	Message        string           `json:"message" validate:"required,min=1,max=500" example:"João Silva foi adicionado à organização TechCorp"`
	Type           NotificationType `json:"type" validate:"required,oneof=info success warning error" example:"info"`
	Data           string           `json:"data,omitempty" example:"{\"action\":\"member_joined\",\"member_id\":\"123\"}"`
}

// NotificationListResponse representa uma lista paginada de notificações
// @Description Resposta da lista de notificações com paginação
type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Total         int64                  `json:"total" example:"25"`
	Page          int                    `json:"page" example:"1"`
	Limit         int                    `json:"limit" example:"10"`
	TotalPages    int                    `json:"total_pages" example:"3"`
}

// UnreadCountResponse representa a contagem de notificações não lidas
// @Description Resposta da contagem de notificações não lidas
type UnreadCountResponse struct {
	Count int64 `json:"count" example:"5"`
}
