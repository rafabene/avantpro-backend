package repositories

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// OrganizationRepositoryInterface defines the interface for organization repository
type OrganizationRepositoryInterface interface {
	// Organization CRUD
	Create(org *models.Organization) error
	GetByID(id uuid.UUID) (*models.Organization, error)
	GetByCreator(creatorID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.Organization, int64, error)
	Update(org *models.Organization) error
	Delete(id uuid.UUID) error
	List(limit, offset int, sortBy, sortOrder string) ([]models.Organization, int64, error)

	// Organization Members
	AddMember(member *models.OrganizationMember) error
	GetMember(orgID, userID uuid.UUID) (*models.OrganizationMember, error)
	GetMembers(orgID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error)
	UpdateMember(member *models.OrganizationMember) error
	RemoveMember(orgID, userID uuid.UUID) error
	GetUserMemberships(userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error)

	// Organization Invites
	CreateInvite(invite *models.OrganizationInvite) error
	GetInviteByToken(token string) (*models.OrganizationInvite, error)
	GetInviteByID(id uuid.UUID) (*models.OrganizationInvite, error)
	GetInvites(orgID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationInvite, int64, error)
	GetPendingInviteByEmail(orgID uuid.UUID, email string) (*models.OrganizationInvite, error)
	UpdateInvite(invite *models.OrganizationInvite) error
	DeleteInvite(id uuid.UUID) error
	ExpireInvites() error
	RegenerateInviteToken(invite *models.OrganizationInvite) error
}

// OrganizationRepository implements the organization repository interface
type OrganizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *gorm.DB) OrganizationRepositoryInterface {
	return &OrganizationRepository{db: db}
}

// Organization CRUD Methods

// Create creates a new organization
func (r *OrganizationRepository) Create(org *models.Organization) error {
	return r.db.Create(org).Error
}

// GetByID retrieves an organization by ID with related data
func (r *OrganizationRepository) GetByID(id uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	err := r.db.Preload("Creator").
		Preload("Members").
		Preload("Members.User").
		Preload("Invites").
		Preload("Invites.Inviter").
		First(&org, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &org, nil
}

// GetByCreator retrieves organizations created by a user
func (r *OrganizationRepository) GetByCreator(creatorID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.Organization, int64, error) {
	var orgs []models.Organization
	var total int64

	query := r.db.Model(&models.Organization{}).Where("created_by = ?", creatorID)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderClause := r.buildOrderClause(sortBy, sortOrder, "organizations")
	query = query.Order(orderClause)

	// Apply pagination and fetch records
	err := query.Preload("Creator").
		Preload("Members").
		Preload("Members.User").
		Limit(limit).
		Offset(offset).
		Find(&orgs).Error

	return orgs, total, err
}

// Update updates an organization
func (r *OrganizationRepository) Update(org *models.Organization) error {
	return r.db.Save(org).Error
}

// Delete soft deletes an organization
func (r *OrganizationRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Organization{}, "id = ?", id).Error
}

// List retrieves all organizations with pagination
func (r *OrganizationRepository) List(limit, offset int, sortBy, sortOrder string) ([]models.Organization, int64, error) {
	var orgs []models.Organization
	var total int64

	query := r.db.Model(&models.Organization{})

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderClause := r.buildOrderClause(sortBy, sortOrder, "organizations")
	query = query.Order(orderClause)

	// Apply pagination and fetch records
	err := query.Preload("Creator").
		Preload("Members").
		Preload("Members.User").
		Limit(limit).
		Offset(offset).
		Find(&orgs).Error

	return orgs, total, err
}

// Organization Members Methods

// AddMember adds a user to an organization
func (r *OrganizationRepository) AddMember(member *models.OrganizationMember) error {
	err := r.db.Create(member).Error
	if err != nil {
		// Check if it's a unique constraint violation (duplicate member)
		if strings.Contains(err.Error(), "idx_org_user") || strings.Contains(err.Error(), "duplicate") {
			return fmt.Errorf("user is already a member of this organization")
		}
		return err
	}
	return nil
}

// GetMember retrieves a specific organization member
func (r *OrganizationRepository) GetMember(orgID, userID uuid.UUID) (*models.OrganizationMember, error) {
	var member models.OrganizationMember
	err := r.db.Preload("User").
		Preload("Organization").
		First(&member, "organization_id = ? AND user_id = ?", orgID, userID).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}

// GetMembers retrieves all members of an organization
func (r *OrganizationRepository) GetMembers(orgID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error) {
	var members []models.OrganizationMember
	var total int64

	query := r.db.Model(&models.OrganizationMember{}).Where("organization_id = ?", orgID)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderClause := r.buildOrderClause(sortBy, sortOrder, "organization_members")
	query = query.Order(orderClause)

	// Apply pagination and fetch records
	err := query.Preload("User").
		Preload("Organization").
		Limit(limit).
		Offset(offset).
		Find(&members).Error

	return members, total, err
}

// UpdateMember updates an organization member
func (r *OrganizationRepository) UpdateMember(member *models.OrganizationMember) error {
	return r.db.Save(member).Error
}

// RemoveMember removes a user from an organization
func (r *OrganizationRepository) RemoveMember(orgID, userID uuid.UUID) error {
	return r.db.Delete(&models.OrganizationMember{}, "organization_id = ? AND user_id = ?", orgID, userID).Error
}

// GetUserMemberships retrieves all organizations a user is a member of
func (r *OrganizationRepository) GetUserMemberships(userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error) {
	var memberships []models.OrganizationMember
	var total int64

	query := r.db.Model(&models.OrganizationMember{}).Where("user_id = ?", userID)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderClause := r.buildOrderClause(sortBy, sortOrder, "organization_members")
	query = query.Order(orderClause)

	// Apply pagination and fetch records
	err := query.Preload("User").
		Preload("Organization").
		Limit(limit).
		Offset(offset).
		Find(&memberships).Error

	return memberships, total, err
}

// Organization Invites Methods

// CreateInvite creates a new organization invitation
func (r *OrganizationRepository) CreateInvite(invite *models.OrganizationInvite) error {
	// Generate unique token
	token, err := r.generateInviteToken()
	if err != nil {
		return err
	}
	invite.Token = token

	// Set expiration (7 days from now)
	invite.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)

	return r.db.Create(invite).Error
}

// GetInviteByToken retrieves an invitation by token
func (r *OrganizationRepository) GetInviteByToken(token string) (*models.OrganizationInvite, error) {
	var invite models.OrganizationInvite
	err := r.db.Preload("Organization").
		Preload("Inviter").
		First(&invite, "token = ?", token).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invite, nil
}

// GetInviteByID retrieves an invitation by ID
func (r *OrganizationRepository) GetInviteByID(id uuid.UUID) (*models.OrganizationInvite, error) {
	var invite models.OrganizationInvite
	err := r.db.Preload("Organization").
		Preload("Inviter").
		First(&invite, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invite, nil
}

// GetInvites retrieves all invitations for an organization
func (r *OrganizationRepository) GetInvites(orgID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationInvite, int64, error) {
	var invites []models.OrganizationInvite
	var total int64

	query := r.db.Model(&models.OrganizationInvite{}).Where("organization_id = ?", orgID)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderClause := r.buildOrderClause(sortBy, sortOrder, "organization_invites")
	query = query.Order(orderClause)

	// Apply pagination and fetch records
	err := query.Preload("Organization").
		Preload("Inviter").
		Limit(limit).
		Offset(offset).
		Find(&invites).Error

	return invites, total, err
}

// GetPendingInviteByEmail retrieves a pending invitation by email for an organization
func (r *OrganizationRepository) GetPendingInviteByEmail(orgID uuid.UUID, email string) (*models.OrganizationInvite, error) {
	var invite models.OrganizationInvite
	err := r.db.First(&invite, "organization_id = ? AND email = ? AND status = ?",
		orgID, email, models.InviteStatusPending).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invite, nil
}

// UpdateInvite updates an organization invitation
func (r *OrganizationRepository) UpdateInvite(invite *models.OrganizationInvite) error {
	return r.db.Model(invite).Select("status", "accepted_at", "updated_at").Updates(invite).Error
}

// DeleteInvite deletes an organization invitation
func (r *OrganizationRepository) DeleteInvite(id uuid.UUID) error {
	return r.db.Delete(&models.OrganizationInvite{}, "id = ?", id).Error
}

// ExpireInvites marks expired invitations as expired
func (r *OrganizationRepository) ExpireInvites() error {
	return r.db.Model(&models.OrganizationInvite{}).
		Where("expires_at < ? AND status = ?", time.Now(), models.InviteStatusPending).
		Update("status", models.InviteStatusExpired).Error
}

// RegenerateInviteToken generates a new token and extends expiry for an invitation
func (r *OrganizationRepository) RegenerateInviteToken(invite *models.OrganizationInvite) error {
	// Generate new token
	token, err := r.generateInviteToken()
	if err != nil {
		return err
	}

	// Set new token and extend expiration (7 days from now)
	invite.Token = token
	invite.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)

	// Update only token and expiry fields
	return r.db.Model(invite).Select("token", "expires_at", "updated_at").Updates(invite).Error
}

// Helper Methods

// buildOrderClause builds the ORDER BY clause for queries
func (r *OrganizationRepository) buildOrderClause(sortBy, sortOrder, tableName string) string {
	// Allowed fields for sorting per table
	allowedFields := map[string][]string{
		"organizations":        {"name", "created_at", "updated_at"},
		"organization_members": {"joined_at", "created_at", "updated_at", "role"},
		"organization_invites": {"email", "status", "created_at", "updated_at", "expires_at"},
	}

	// Validate sortBy field
	if fields, ok := allowedFields[tableName]; ok {
		validField := false
		for _, field := range fields {
			if field == sortBy {
				validField = true
				break
			}
		}
		if !validField {
			sortBy = "created_at" // Default field
		}
	} else {
		sortBy = "created_at" // Default field
	}

	// Validate sortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc" // Default order
	}

	// Convert camelCase to snake_case
	sortBy = r.camelToSnake(sortBy)

	return fmt.Sprintf("%s %s", sortBy, sortOrder)
}

// camelToSnake converts camelCase to snake_case
func (r *OrganizationRepository) camelToSnake(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// generateInviteToken generates a unique token for invitations
func (r *OrganizationRepository) generateInviteToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
