package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationRole represents the role of a user in an organization
type OrganizationRole string

const (
	OrganizationRoleAdmin OrganizationRole = "admin"
	OrganizationRoleUser  OrganizationRole = "user"
)

// Organization represents an organization entity
// @Description Organization information
type Organization struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string         `json:"name" gorm:"not null" validate:"required,min=2,max=100" example:"My Company"`
	Description string         `json:"description" gorm:"type:text" validate:"max=500" example:"A great company"`
	CreatedBy   uuid.UUID      `json:"created_by" gorm:"type:uuid;not null" example:"550e8400-e29b-41d4-a716-446655440001"`
	Creator     User           `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Members     []OrganizationMember `json:"members,omitempty" gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Invites     []OrganizationInvite `json:"invites,omitempty" gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt   time.Time      `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time      `json:"updated_at" example:"2023-01-01T12:00:00Z"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// OrganizationMember represents a member of an organization
// @Description Organization member information
type OrganizationMember struct {
	ID             uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationID uuid.UUID        `json:"organization_id" gorm:"type:uuid;not null;uniqueIndex:idx_org_user" example:"550e8400-e29b-41d4-a716-446655440001"`
	UserID         uuid.UUID        `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_org_user" example:"550e8400-e29b-41d4-a716-446655440002"`
	Role           OrganizationRole `json:"role" gorm:"type:varchar(50);not null;default:'user'" example:"user"`
	Organization   Organization     `json:"organization,omitempty" gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User           User             `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	JoinedAt       time.Time        `json:"joined_at" gorm:"autoCreateTime" example:"2023-01-01T12:00:00Z"`
	CreatedAt      time.Time        `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt      time.Time        `json:"updated_at" example:"2023-01-01T12:00:00Z"`
	DeletedAt      gorm.DeletedAt   `json:"-" gorm:"index"`
}

// OrganizationInvite represents an invitation to join an organization
// @Description Organization invitation information
type OrganizationInvite struct {
	ID             uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationID uuid.UUID        `json:"organization_id" gorm:"type:uuid;not null" example:"550e8400-e29b-41d4-a716-446655440001"`
	Email          string           `json:"email" gorm:"not null" validate:"required,email" example:"user@example.com"`
	Role           OrganizationRole `json:"role" gorm:"type:varchar(50);not null;default:'user'" example:"user"`
	InvitedBy      uuid.UUID        `json:"invited_by" gorm:"type:uuid;not null" example:"550e8400-e29b-41d4-a716-446655440002"`
	Token          string           `json:"token" gorm:"uniqueIndex;not null" example:"invitation-token-123"`
	Status         InviteStatus     `json:"status" gorm:"type:varchar(20);not null;default:'pending'" example:"pending"`
	ExpiresAt      time.Time        `json:"expires_at" gorm:"not null" example:"2023-12-31T23:59:59Z"`
	AcceptedAt     *time.Time       `json:"accepted_at,omitempty" example:"2023-01-01T12:00:00Z"`
	Organization   Organization     `json:"organization,omitempty" gorm:"foreignKey:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Inviter        User             `json:"inviter,omitempty" gorm:"foreignKey:InvitedBy;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	CreatedAt      time.Time        `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt      time.Time        `json:"updated_at" example:"2023-01-01T12:00:00Z"`
	DeletedAt      gorm.DeletedAt   `json:"-" gorm:"index"`
}

// InviteStatus represents the status of an organization invitation
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// OrganizationCreateRequest represents the request body for creating an organization
// @Description Organization creation request
type OrganizationCreateRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100" example:"My Company"`
	Description string `json:"description" validate:"max=500" example:"A great company"`
}

// OrganizationUpdateRequest represents the request body for updating an organization
// @Description Organization update request
type OrganizationUpdateRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=100" example:"My Company"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500" example:"A great company"`
}

// OrganizationInviteRequest represents the request body for inviting a user to an organization
// @Description Organization invitation request
type OrganizationInviteRequest struct {
	Email string           `json:"email" validate:"required,email" example:"user@example.com"`
	Role  OrganizationRole `json:"role" validate:"required,oneof=admin user" example:"user"`
}

// OrganizationMemberUpdateRequest represents the request body for updating a member's role
// @Description Organization member update request
type OrganizationMemberUpdateRequest struct {
	Role OrganizationRole `json:"role" validate:"required,oneof=admin user" example:"user"`
}

// OrganizationResponse represents the response body for organization operations
// @Description Organization response
type OrganizationResponse struct {
	ID          uuid.UUID                      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string                         `json:"name" example:"My Company"`
	Description string                         `json:"description" example:"A great company"`
	CreatedBy   uuid.UUID                      `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440001"`
	Creator     *UserResponse                  `json:"creator,omitempty"`
	Members     []OrganizationMemberResponse   `json:"members,omitempty"`
	Invites     []OrganizationInviteResponse   `json:"invites,omitempty"`
	CreatedAt   time.Time                      `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time                      `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// OrganizationMemberResponse represents the response body for organization member operations
// @Description Organization member response
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

// OrganizationInviteResponse represents the response body for organization invite operations
// @Description Organization invitation response
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

// OrganizationListResponse represents the response body for listing organizations
// @Description Organization list response with pagination
type OrganizationListResponse struct {
	Data  []OrganizationResponse `json:"data"`
	Total int64                  `json:"total" example:"100"`
	Limit int                    `json:"limit" example:"50"`
	Page  int                    `json:"page" example:"1"`
}

// OrganizationMemberListResponse represents the response body for listing organization members
// @Description Organization member list response with pagination
type OrganizationMemberListResponse struct {
	Data  []OrganizationMemberResponse `json:"data"`
	Total int64                        `json:"total" example:"100"`
	Limit int                          `json:"limit" example:"50"`
	Page  int                          `json:"page" example:"1"`
}

// OrganizationInviteListResponse represents the response body for listing organization invites
// @Description Organization invitation list response with pagination
type OrganizationInviteListResponse struct {
	Data  []OrganizationInviteResponse `json:"data"`
	Total int64                        `json:"total" example:"100"`
	Limit int                          `json:"limit" example:"50"`
	Page  int                          `json:"page" example:"1"`
}

// IsExpired checks if the invitation is expired
func (oi *OrganizationInvite) IsExpired() bool {
	return time.Now().After(oi.ExpiresAt)
}

// IsAdmin checks if the member has admin role
func (om *OrganizationMember) IsAdmin() bool {
	return om.Role == OrganizationRoleAdmin
}

// BeforeCreate is a GORM hook that runs before creating an organization
func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	return nil
}

// AfterCreate is a GORM hook that runs after creating an organization
func (o *Organization) AfterCreate(tx *gorm.DB) error {
	// Automatically add the creator as an admin member
	member := &OrganizationMember{
		OrganizationID: o.ID,
		UserID:         o.CreatedBy,
		Role:           OrganizationRoleAdmin,
	}
	return tx.Create(member).Error
}