package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
)

// Notification represents a notification entity
// @Description Notification information
type Notification struct {
	ID             uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID         uuid.UUID        `json:"user_id" gorm:"type:uuid;not null;index" example:"550e8400-e29b-41d4-a716-446655440001"`
	OrganizationID uuid.UUID        `json:"organization_id" gorm:"type:uuid;not null;index" example:"550e8400-e29b-41d4-a716-446655440002"`
	Title          string           `json:"title" gorm:"not null" validate:"required,min=1,max=200" example:"Novo membro na organização"`
	Message        string           `json:"message" gorm:"not null" validate:"required,min=1,max=500" example:"João Silva foi adicionado à organização TechCorp"`
	Type           NotificationType `json:"type" gorm:"not null" validate:"required,oneof=info success warning error" example:"info"`
	Read           bool             `json:"read" gorm:"default:false" example:"false"`
	ReadAt         *time.Time       `json:"read_at,omitempty" example:"2023-01-01T12:00:00Z"`
	Data           string           `json:"data,omitempty" gorm:"type:text" example:"{\"action\":\"member_joined\",\"member_id\":\"123\"}"`
	CreatedAt      time.Time        `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt      time.Time        `json:"updated_at" example:"2023-01-01T12:00:00Z"`
	DeletedAt      gorm.DeletedAt   `json:"-" gorm:"index"`

	// Associations
	User         *User         `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Organization *Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// NotificationResponse represents the response body for notification operations
// @Description Notification response
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

// CreateNotificationRequest represents the request body for creating a notification
// @Description Create notification request
type CreateNotificationRequest struct {
	UserID         uuid.UUID        `json:"user_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	OrganizationID uuid.UUID        `json:"organization_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440002"`
	Title          string           `json:"title" validate:"required,min=1,max=200" example:"Novo membro na organização"`
	Message        string           `json:"message" validate:"required,min=1,max=500" example:"João Silva foi adicionado à organização TechCorp"`
	Type           NotificationType `json:"type" validate:"required,oneof=info success warning error" example:"info"`
	Data           string           `json:"data,omitempty" example:"{\"action\":\"member_joined\",\"member_id\":\"123\"}"`
}

// MarkNotificationReadRequest represents the request body for marking notification as read
// @Description Mark notification as read request
type MarkNotificationReadRequest struct {
	Read bool `json:"read" example:"true"`
}

// MarkAsRead marks the notification as read and sets the read timestamp
func (n *Notification) MarkAsRead() {
	if !n.Read {
		n.Read = true
		now := time.Now()
		n.ReadAt = &now
	}
}

// MarkAsUnread marks the notification as unread and clears the read timestamp
func (n *Notification) MarkAsUnread() {
	if n.Read {
		n.Read = false
		n.ReadAt = nil
	}
}
