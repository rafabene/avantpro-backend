package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationRole representa o papel de um usuário em uma organização
type OrganizationRole string

const (
	OrganizationRoleAdmin OrganizationRole = "admin"
	OrganizationRoleUser  OrganizationRole = "user"
)

// Organization representa uma entidade de organização
type Organization struct {
	ID          uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string               `gorm:"not null"`
	Description string               `gorm:"type:text"`
	CreatedBy   uuid.UUID            `gorm:"type:uuid;not null"`
	Creator     User                 `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Members     []OrganizationMember `gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Invites     []OrganizationInvite `gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// OrganizationMember representa um membro de uma organização
type OrganizationMember struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_org_user"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_org_user"`
	Role           OrganizationRole `gorm:"type:varchar(50);not null;default:'user'"`
	Organization   Organization     `gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User           User             `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	JoinedAt       time.Time        `gorm:"autoCreateTime"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// OrganizationInvite representa um convite para ingressar em uma organização
type OrganizationInvite struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID        `gorm:"type:uuid;not null"`
	Email          string           `gorm:"not null"`
	Role           OrganizationRole `gorm:"type:varchar(50);not null;default:'user'"`
	InvitedBy      uuid.UUID        `gorm:"type:uuid;not null"`
	Token          string           `gorm:"uniqueIndex;not null"`
	Status         InviteStatus     `gorm:"type:varchar(20);not null;default:'pending'"`
	ExpiresAt      time.Time        `gorm:"not null"`
	AcceptedAt     *time.Time
	Organization   Organization `gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Inviter        User         `gorm:"foreignKey:InvitedBy;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// InviteStatus representa o status de um convite da organização
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// IsExpired verifica se o convite está expirado
func (oi *OrganizationInvite) IsExpired() bool {
	return time.Now().After(oi.ExpiresAt)
}

// IsAdmin verifica se o membro tem papel de administrador
func (om *OrganizationMember) IsAdmin() bool {
	return om.Role == OrganizationRoleAdmin
}

// BeforeCreate é um hook do GORM que executa antes de criar uma organização
func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	return nil
}

// AfterCreate é um hook do GORM que executa após criar uma organização
func (o *Organization) AfterCreate(tx *gorm.DB) error {
	// Adicionar automaticamente o criador como membro administrador
	member := &OrganizationMember{
		OrganizationID: o.ID,
		UserID:         o.CreatedBy,
		Role:           OrganizationRoleAdmin,
	}
	return tx.Create(member).Error
}
