package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationEvent represents the type of event that triggers a notification
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

// NotificationPreference represents user preferences for notification types
// @Description User notification preferences
type NotificationPreference struct {
	ID        uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    uuid.UUID         `json:"user_id" gorm:"type:uuid;not null;index" example:"550e8400-e29b-41d4-a716-446655440001"`
	Event     NotificationEvent `json:"event" gorm:"not null" validate:"required" example:"member_joined"`
	Enabled   bool              `json:"enabled" gorm:"default:true" example:"true"`
	CreatedAt time.Time         `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time         `json:"updated_at" example:"2023-01-01T12:00:00Z"`
	DeletedAt gorm.DeletedAt    `json:"-" gorm:"index"`

	// Associations
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// NotificationPreferenceResponse represents the response body for notification preference operations
// @Description Notification preference response
type NotificationPreferenceResponse struct {
	ID        uuid.UUID         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Event     NotificationEvent `json:"event" example:"member_joined"`
	Enabled   bool              `json:"enabled" example:"true"`
	CreatedAt time.Time         `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time         `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// NotificationPreferenceUpdateRequest represents the request body for updating notification preferences
// @Description Update notification preferences request
type NotificationPreferenceUpdateRequest struct {
	Enabled *bool `json:"enabled,omitempty" example:"true"`
}

// NotificationPreferenceBulkUpdateRequest represents the request body for bulk updating notification preferences
// @Description Bulk update notification preferences request
type NotificationPreferenceBulkUpdateRequest struct {
	Preferences []NotificationPreferenceBulkItem `json:"preferences" validate:"required,min=1"`
}

// NotificationPreferenceBulkItem represents a single preference item in bulk update
// @Description Single notification preference item for bulk update
type NotificationPreferenceBulkItem struct {
	Event   NotificationEvent `json:"event" validate:"required" example:"member_joined"`
	Enabled bool              `json:"enabled" example:"true"`
}

// TestNotificationRequest represents the request body for generating a test notification
// @Description Test notification request
type TestNotificationRequest struct {
	OrganizationID uuid.UUID        `json:"organization_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440002"`
	Type           NotificationType `json:"type" validate:"required,oneof=info success warning error" example:"info"`
	Title          string           `json:"title" validate:"required,min=1,max=200" example:"Test Notification"`
	Message        string           `json:"message" validate:"required,min=1,max=500" example:"This is a test notification to verify your settings"`
}

// GetDefaultNotificationPreferences returns the default notification preferences for a user
func GetDefaultNotificationPreferences(userID uuid.UUID) []NotificationPreference {
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
			UserID:  userID,
			Event:   event,
			Enabled: true, // All notifications enabled by default
		}
	}

	return preferences
}

// GetEventDescription returns a human-readable description for a notification event
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

// IsValidEvent checks if the event is valid
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
