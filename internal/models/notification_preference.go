package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationEvent representa o tipo de evento que aciona uma notificação
type NotificationEvent string

const (
	NotificationEventMemberJoined       NotificationEvent = "member_joined"
	NotificationEventMemberLeft         NotificationEvent = "member_left"
	NotificationEventMemberRoleChanged  NotificationEvent = "member_role_changed"
	NotificationEventInvitationSent     NotificationEvent = "invitation_sent"
	NotificationEventInvitationAccepted NotificationEvent = "invitation_accepted"
	NotificationEventInvitationExpired  NotificationEvent = "invitation_expired"
	NotificationEventOrganizationUpdate NotificationEvent = "organization_update"
)

// NotificationPreference representa as preferências da organização para tipos de notificação
type NotificationPreference struct {
	ID             uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID         `gorm:"type:uuid;not null;index"`
	Event          NotificationEvent `gorm:"not null"`
	Enabled        bool              `gorm:"default:true"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	// Associações
	Organization *Organization `gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// GetDefaultNotificationPreferences retorna as preferências de notificação padrão para uma organização
func GetDefaultNotificationPreferences(organizationID uuid.UUID) []NotificationPreference {
	events := []NotificationEvent{
		NotificationEventMemberJoined,
		NotificationEventMemberLeft,
		NotificationEventMemberRoleChanged,
		NotificationEventInvitationSent,
		NotificationEventInvitationAccepted,
		NotificationEventInvitationExpired,
		NotificationEventOrganizationUpdate,
	}

	preferences := make([]NotificationPreference, len(events))
	for i, event := range events {
		preferences[i] = NotificationPreference{
			OrganizationID: organizationID,
			Event:          event,
			Enabled:        true,
		}
	}

	return preferences
}

// GetDescription retorna uma descrição legível para um evento de notificação
func (event NotificationEvent) GetDescription() string {
	switch event {
	case NotificationEventMemberJoined:
		return "Quando um novo membro entrar na organização"
	case NotificationEventMemberLeft:
		return "Quando um membro sair da organização"
	case NotificationEventMemberRoleChanged:
		return "Quando o papel de um membro for alterado"
	case NotificationEventInvitationSent:
		return "Quando um convite for enviado"
	case NotificationEventInvitationAccepted:
		return "Quando um convite for aceito"
	case NotificationEventInvitationExpired:
		return "Quando um convite expirar"
	case NotificationEventOrganizationUpdate:
		return "Quando a organização for atualizada"
	default:
		return string(event)
	}
}

// IsValid verifica se o evento é válido
func (event NotificationEvent) IsValid() bool {
	switch event {
	case NotificationEventMemberJoined,
		NotificationEventMemberLeft,
		NotificationEventMemberRoleChanged,
		NotificationEventInvitationSent,
		NotificationEventInvitationAccepted,
		NotificationEventInvitationExpired,
		NotificationEventOrganizationUpdate:
		return true
	default:
		return false
	}
}

// NotificationPreferenceBulkItem representa um item de preferência único na atualização em lote
type NotificationPreferenceBulkItem struct {
	Event   NotificationEvent `json:"event" validate:"required" example:"member_joined"`
	Enabled bool              `json:"enabled" example:"true"`
}
