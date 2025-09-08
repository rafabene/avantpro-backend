package controllers

import (
	"time"

	"github.com/google/uuid"
)

// NotificationPreferenceResponse representa o corpo da resposta para operações de preferência de notificação
// @Description Resposta da preferência de notificação
type NotificationPreferenceResponse struct {
	ID        uuid.UUID         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Event     NotificationEvent `json:"event" example:"member_joined"`
	Enabled   bool              `json:"enabled" example:"true"`
	CreatedAt time.Time         `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time         `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// NotificationPreferenceUpdateRequest representa o corpo da requisição para atualizar preferências de notificação
// @Description Requisição de atualização de preferências de notificação
type NotificationPreferenceUpdateRequest struct {
	Enabled *bool `json:"enabled,omitempty" example:"true"`
}

// NotificationPreferenceBulkUpdateRequest representa o corpo da requisição para atualização em lote de preferências de notificação
// @Description Requisição de atualização em lote de preferências de notificação
type NotificationPreferenceBulkUpdateRequest struct {
	Preferences []NotificationPreferenceBulkItem `json:"preferences" validate:"required,min=1"`
}

// TestNotificationRequest representa o corpo da requisição para gerar uma notificação de teste
// @Description Requisição de notificação de teste
type TestNotificationRequest struct {
	OrganizationID uuid.UUID        `json:"organization_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440002"`
	Type           NotificationType `json:"type" validate:"required,oneof=info success warning error" example:"info"`
	Title          string           `json:"title" validate:"required,min=1,max=200" example:"Test Notification"`
	Message        string           `json:"message" validate:"required,min=1,max=500" example:"This is a test notification to verify your settings"`
}

// NotificationPreferenceListResponse representa uma lista de preferências de notificação
// @Description Resposta da lista de preferências de notificação
type NotificationPreferenceListResponse struct {
	Preferences []NotificationPreferenceResponse `json:"preferences"`
}

// NotificationEventsResponse representa os eventos de notificação disponíveis
// @Description Resposta dos eventos de notificação disponíveis
type NotificationEventsResponse struct {
	Events []NotificationEvent `json:"events"`
}

// TestNotificationResponse representa a resposta após enviar uma notificação de teste
// @Description Resposta da notificação de teste
type TestNotificationResponse struct {
	Message      string               `json:"message" example:"Test notification sent successfully"`
	Notification NotificationResponse `json:"notification"`
}

// Tipos de conversão para services (usados internamente nos controllers)
type ServiceNotificationPreferenceUpdateRequest struct {
	Enabled *bool
}

type ServiceNotificationPreferenceBulkUpdateRequest struct {
	Preferences []ServiceNotificationPreferenceBulkItem
}

type ServiceNotificationPreferenceBulkItem struct {
	Event   NotificationEvent
	Enabled bool
}

type ServiceTestNotificationRequest struct {
	OrganizationID uuid.UUID
	Type           NotificationType
	Title          string
	Message        string
}
