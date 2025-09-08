package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType representa o tipo de notificação
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
)

// Notification representa uma entidade de notificação
type Notification struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null;index"`
	OrganizationID uuid.UUID        `gorm:"type:uuid;not null;index"`
	Title          string           `gorm:"not null"`
	Message        string           `gorm:"not null"`
	Type           NotificationType `gorm:"not null"`
	Read           bool             `gorm:"default:false"`
	ReadAt         *time.Time
	Data           string `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	// Associações
	User         *User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Organization *Organization `gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// MarkAsRead marca a notificação como lida e define o timestamp de leitura
func (n *Notification) MarkAsRead() {
	if !n.Read {
		n.Read = true
		now := time.Now()
		n.ReadAt = &now
	}
}

// MarkAsUnread marca a notificação como não lida e limpa o timestamp de leitura
func (n *Notification) MarkAsUnread() {
	if n.Read {
		n.Read = false
		n.ReadAt = nil
	}
}
