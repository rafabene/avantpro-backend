package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/repositories"
)

// OrganizationServiceInterface defines the interface for organization service operations.
// This interface provides methods for managing organizations, their members, and invitations.
type OrganizationServiceInterface interface {
	// Organization CRUD Operations

	// CreateOrganization creates a new organization with the specified creator as admin.
	// The creator is automatically added as an admin member of the organization.
	// Parameters:
	//   - req: Organization creation request containing name and description
	//   - creatorID: UUID of the user creating the organization
	// Returns:
	//   - *models.Organization: The created organization with all related data
	//   - error: Error if creation fails or creator doesn't exist
	CreateOrganization(req *models.OrganizationCreateRequest, creatorID uuid.UUID) (*models.Organization, error)

	// GetOrganization retrieves an organization by its ID.
	// This method loads the organization with all related data (creator, members, invites).
	// Parameters:
	//   - id: UUID of the organization to retrieve
	// Returns:
	//   - *models.Organization: The organization with related data
	//   - error: Error if organization not found or database error
	GetOrganization(id uuid.UUID) (*models.Organization, error)

	// GetUserOrganizations retrieves all organizations created by a specific user.
	// Results are paginated and can be sorted by various fields.
	// Parameters:
	//   - userID: UUID of the user whose organizations to retrieve
	//   - limit: Maximum number of results to return
	//   - offset: Number of results to skip (for pagination)
	//   - sortBy: Field to sort by (name, created_at, updated_at)
	//   - sortOrder: Sort direction (asc, desc)
	// Returns:
	//   - []models.Organization: List of organizations
	//   - int64: Total count of organizations (for pagination)
	//   - error: Error if query fails
	GetUserOrganizations(userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.Organization, int64, error)

	// UpdateOrganization updates an existing organization's details.
	// Only admin members can update organization information.
	// Parameters:
	//   - id: UUID of the organization to update
	//   - req: Update request containing new name and/or description
	//   - userID: UUID of the user requesting the update (must be admin)
	// Returns:
	//   - *models.Organization: The updated organization
	//   - error: Error if user lacks permissions or update fails
	UpdateOrganization(id uuid.UUID, req *models.OrganizationUpdateRequest, userID uuid.UUID) (*models.Organization, error)

	// DeleteOrganization soft-deletes an organization.
	// Only the original creator can delete an organization.
	// This will also cascade delete all members and invitations.
	// Parameters:
	//   - id: UUID of the organization to delete
	//   - userID: UUID of the user requesting deletion (must be creator)
	// Returns:
	//   - error: Error if user lacks permissions or deletion fails
	DeleteOrganization(id uuid.UUID, userID uuid.UUID) error

	// Organization Member Management

	// GetOrganizationMembers retrieves all members of an organization.
	// Only existing members can view the member list.
	// Results are paginated and sortable.
	// Parameters:
	//   - orgID: UUID of the organization
	//   - userID: UUID of the requesting user (must be a member)
	//   - limit: Maximum number of results to return
	//   - offset: Number of results to skip (for pagination)
	//   - sortBy: Field to sort by (joined_at, role, created_at, updated_at)
	//   - sortOrder: Sort direction (asc, desc)
	// Returns:
	//   - []models.OrganizationMember: List of members with user details
	//   - int64: Total count of members (for pagination)
	//   - error: Error if user lacks permissions or query fails
	GetOrganizationMembers(orgID uuid.UUID, userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error)

	// UpdateMemberRole updates a member's role within an organization.
	// Only admin members can change roles of other members.
	// The organization creator's role cannot be changed from admin.
	// Parameters:
	//   - orgID: UUID of the organization
	//   - memberUserID: UUID of the member whose role to update
	//   - req: Update request containing the new role (admin or user)
	//   - requestorID: UUID of the user making the request (must be admin)
	// Returns:
	//   - *models.OrganizationMember: The updated member with new role
	//   - error: Error if user lacks permissions or update fails
	UpdateMemberRole(orgID uuid.UUID, memberUserID uuid.UUID, req *models.OrganizationMemberUpdateRequest, requestorID uuid.UUID) (*models.OrganizationMember, error)

	// RemoveMember removes a member from an organization.
	// Admin members can remove any other member (except the creator).
	// Regular members can only remove themselves (leave the organization).
	// The organization creator cannot be removed.
	// Parameters:
	//   - orgID: UUID of the organization
	//   - memberUserID: UUID of the member to remove
	//   - requestorID: UUID of the user making the request
	// Returns:
	//   - error: Error if user lacks permissions or removal fails
	RemoveMember(orgID uuid.UUID, memberUserID uuid.UUID, requestorID uuid.UUID) error

	// GetUserMemberships retrieves all organizations a user is a member of.
	// This includes organizations where the user is either admin or regular member.
	// Results are paginated and sortable.
	// Parameters:
	//   - userID: UUID of the user whose memberships to retrieve
	//   - limit: Maximum number of results to return
	//   - offset: Number of results to skip (for pagination)
	//   - sortBy: Field to sort by (joined_at, role, created_at, updated_at)
	//   - sortOrder: Sort direction (asc, desc)
	// Returns:
	//   - []models.OrganizationMember: List of memberships with organization details
	//   - int64: Total count of memberships (for pagination)
	//   - error: Error if query fails
	GetUserMemberships(userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error)

	// Organization Invitation Management

	// InviteUser sends an invitation for a user to join an organization.
	// Only admin members can send invitations.
	// The system checks if the user is already a member or has a pending invitation.
	// An email invitation is automatically sent to the specified email address.
	// Parameters:
	//   - orgID: UUID of the organization
	//   - req: Invitation request containing email and role for the invitee
	//   - inviterID: UUID of the user sending the invitation (must be admin)
	// Returns:
	//   - *models.OrganizationInvite: The created invitation with token and expiry
	//   - error: Error if user lacks permissions, already invited, or email sending fails
	InviteUser(orgID uuid.UUID, req *models.OrganizationInviteRequest, inviterID uuid.UUID) (*models.OrganizationInvite, error)

	// GetOrganizationInvites retrieves all pending invitations for an organization.
	// Only admin members can view the organization's invitations.
	// Results are paginated and sortable.
	// Parameters:
	//   - orgID: UUID of the organization
	//   - userID: UUID of the requesting user (must be admin)
	//   - limit: Maximum number of results to return
	//   - offset: Number of results to skip (for pagination)
	//   - sortBy: Field to sort by (email, status, created_at, updated_at, expires_at)
	//   - sortOrder: Sort direction (asc, desc)
	// Returns:
	//   - []models.OrganizationInvite: List of invitations with details
	//   - int64: Total count of invitations (for pagination)
	//   - error: Error if user lacks permissions or query fails
	GetOrganizationInvites(orgID uuid.UUID, userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationInvite, int64, error)

	// AcceptInvite accepts an organization invitation using the invitation token.
	// The invitation must be valid, not expired, and the email must match the user's email.
	// Upon acceptance, the user becomes a member with the role specified in the invitation.
	// Parameters:
	//   - token: Unique invitation token from the email link
	//   - userID: UUID of the user accepting the invitation
	// Returns:
	//   - *models.OrganizationMember: The created membership record
	//   - error: Error if token invalid, expired, or email mismatch
	AcceptInvite(token string, userID uuid.UUID) (*models.OrganizationMember, error)

	// ValidateInvite validates an organization invitation and checks if the invited user exists.
	// This method does not require authentication and can be used to determine
	// whether a user should be redirected to login or registration.
	// Returns:
	//   - map[string]interface{}: Contains validation results and user existence info
	//   - error: Error if invitation is invalid, expired, or not found
	ValidateInvite(token string) (map[string]interface{}, error)

	// RevokeInvite cancels a pending organization invitation.
	// Only admin members can revoke invitations.
	// This prevents the invitation from being accepted and marks it as revoked.
	// Parameters:
	//   - inviteID: UUID of the invitation to revoke
	//   - userID: UUID of the user revoking the invitation (must be admin)
	// Returns:
	//   - error: Error if user lacks permissions or revocation fails
	RevokeInvite(inviteID uuid.UUID, userID uuid.UUID) error

	// ResendInvite resends an organization invitation by generating a new token and extending expiry.
	// Only admin members can resend invitations.
	// This creates a new invitation token and sends a new email notification.
	// Parameters:
	//   - inviteID: UUID of the invitation to resend
	//   - userID: UUID of the user requesting the resend (must be admin)
	// Returns:
	//   - *models.OrganizationInvite: The updated invitation with new token and expiry
	//   - error: Error if user lacks permissions or resend fails
	ResendInvite(inviteID uuid.UUID, userID uuid.UUID) (*models.OrganizationInvite, error)
}

// OrganizationService implements the organization service interface.
// It provides business logic for organization management including CRUD operations,
// member management, and invitation handling. This service coordinates between
// the organization repository, user repository, and email service to provide
// complete organization functionality with proper permission checks.
type OrganizationService struct {
	orgRepo             repositories.OrganizationRepositoryInterface  // Repository for organization data operations
	userRepo            repositories.UserRepository                   // Repository for user data operations
	emailService        EmailServiceInterface                         // Service for sending email notifications
	notificationService NotificationService                           // Service for managing notifications
	preferenceRepo      repositories.NotificationPreferenceRepository // Repository for notification preferences
}

// NewOrganizationService creates a new instance of OrganizationService.
// This constructor initializes the service with all required dependencies.
// Parameters:
//   - orgRepo: Repository interface for organization data operations
//   - userRepo: Repository interface for user data operations
//   - emailService: Service interface for sending emails
//   - notificationService: Service for managing notifications
//   - preferenceRepo: Repository for notification preferences
//
// Returns:
//   - OrganizationServiceInterface: Configured organization service ready for use
func NewOrganizationService(orgRepo repositories.OrganizationRepositoryInterface, userRepo repositories.UserRepository, emailService EmailServiceInterface, notificationService NotificationService, preferenceRepo repositories.NotificationPreferenceRepository) OrganizationServiceInterface {
	return &OrganizationService{
		orgRepo:             orgRepo,
		userRepo:            userRepo,
		emailService:        emailService,
		notificationService: notificationService,
		preferenceRepo:      preferenceRepo,
	}
}

// Organization CRUD Methods

// CreateOrganization creates a new organization with the creator as admin
func (s *OrganizationService) CreateOrganization(req *models.OrganizationCreateRequest, creatorID uuid.UUID) (*models.Organization, error) {
	// Validate creator exists
	creator, err := s.userRepo.GetByID(creatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator: %w", err)
	}
	if creator == nil {
		return nil, fmt.Errorf("creator not found")
	}

	// Create organization
	org := &models.Organization{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   creatorID,
	}

	if err := s.orgRepo.Create(org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Create default notification preferences for the organization
	if err := s.preferenceRepo.CreateDefaults(org.ID); err != nil {
		return nil, fmt.Errorf("failed to create default notification preferences: %w", err)
	}

	// Fetch the created organization with relations
	return s.orgRepo.GetByID(org.ID)
}

// GetOrganization retrieves an organization by ID with permission check
func (s *OrganizationService) GetOrganization(id uuid.UUID) (*models.Organization, error) {
	org, err := s.orgRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, fmt.Errorf("organization not found")
	}

	return org, nil
}

// GetUserOrganizations retrieves all organizations created by a user
func (s *OrganizationService) GetUserOrganizations(userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.Organization, int64, error) {
	return s.orgRepo.GetByCreator(userID, limit, offset, sortBy, sortOrder)
}

// UpdateOrganization updates an organization (only admin can update)
func (s *OrganizationService) UpdateOrganization(id uuid.UUID, req *models.OrganizationUpdateRequest, userID uuid.UUID) (*models.Organization, error) {
	// Get organization
	org, err := s.orgRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, fmt.Errorf("organization not found")
	}

	// Check if user is admin
	if !s.isUserAdmin(org, userID) {
		return nil, fmt.Errorf("insufficient permissions: only admins can update organization")
	}

	// Update fields
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Description != nil {
		org.Description = *req.Description
	}

	if err := s.orgRepo.Update(org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return s.orgRepo.GetByID(org.ID)
}

// DeleteOrganization deletes an organization (only creator can delete)
func (s *OrganizationService) DeleteOrganization(id uuid.UUID, userID uuid.UUID) error {
	// Get organization
	org, err := s.orgRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return fmt.Errorf("organization not found")
	}

	// Check if user is the creator
	if org.CreatedBy != userID {
		return fmt.Errorf("insufficient permissions: only the creator can delete organization")
	}

	return s.orgRepo.Delete(id)
}

// Organization Members Methods

// GetOrganizationMembers retrieves members of an organization (members can view)
func (s *OrganizationService) GetOrganizationMembers(orgID uuid.UUID, userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error) {
	// Check if user is a member of the organization
	member, err := s.orgRepo.GetMember(orgID, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to check membership: %w", err)
	}
	if member == nil {
		return nil, 0, fmt.Errorf("insufficient permissions: user is not a member of this organization")
	}

	return s.orgRepo.GetMembers(orgID, limit, offset, sortBy, sortOrder)
}

// UpdateMemberRole updates a member's role (only admin can update)
func (s *OrganizationService) UpdateMemberRole(orgID uuid.UUID, memberUserID uuid.UUID, req *models.OrganizationMemberUpdateRequest, requestorID uuid.UUID) (*models.OrganizationMember, error) {
	// Get organization
	org, err := s.orgRepo.GetByID(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, fmt.Errorf("organization not found")
	}

	// Check if requestor is admin
	if !s.isUserAdmin(org, requestorID) {
		return nil, fmt.Errorf("insufficient permissions: only admins can update member roles")
	}

	// Get member
	member, err := s.orgRepo.GetMember(orgID, memberUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return nil, fmt.Errorf("member not found")
	}

	// Prevent removing admin role from creator
	if member.UserID == org.CreatedBy && req.Role != models.OrganizationRoleAdmin {
		return nil, fmt.Errorf("cannot change role of organization creator")
	}

	// Update role
	member.Role = req.Role
	if err := s.orgRepo.UpdateMember(member); err != nil {
		return nil, fmt.Errorf("failed to update member role: %w", err)
	}

	return s.orgRepo.GetMember(orgID, memberUserID)
}

// RemoveMember removes a member from an organization (admin can remove, users can leave)
func (s *OrganizationService) RemoveMember(orgID uuid.UUID, memberUserID uuid.UUID, requestorID uuid.UUID) error {
	// Get organization
	org, err := s.orgRepo.GetByID(orgID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return fmt.Errorf("organization not found")
	}

	// Prevent removing the creator
	if memberUserID == org.CreatedBy {
		return fmt.Errorf("cannot remove organization creator")
	}

	// Check permissions: admin can remove anyone, user can only remove themselves
	if requestorID != memberUserID && !s.isUserAdmin(org, requestorID) {
		return fmt.Errorf("insufficient permissions: only admins can remove other members")
	}

	// Check if member exists
	member, err := s.orgRepo.GetMember(orgID, memberUserID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return fmt.Errorf("member not found")
	}

	return s.orgRepo.RemoveMember(orgID, memberUserID)
}

// GetUserMemberships retrieves all organizations a user is a member of
func (s *OrganizationService) GetUserMemberships(userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error) {
	return s.orgRepo.GetUserMemberships(userID, limit, offset, sortBy, sortOrder)
}

// Organization Invites Methods

// InviteUser sends an invitation to join an organization (only admin can invite)
func (s *OrganizationService) InviteUser(orgID uuid.UUID, req *models.OrganizationInviteRequest, inviterID uuid.UUID) (*models.OrganizationInvite, error) {
	// Get organization
	org, err := s.orgRepo.GetByID(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, fmt.Errorf("organization not found")
	}

	// Check if inviter is admin
	if !s.isUserAdmin(org, inviterID) {
		return nil, fmt.Errorf("insufficient permissions: only admins can send invitations")
	}

	// If user exists, check if they're already a member
	existingUser, err := s.userRepo.GetByUsername(req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		existingMember, err := s.orgRepo.GetMember(orgID, existingUser.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing membership: %w", err)
		}
		if existingMember != nil && existingMember.DeletedAt.Time.IsZero() {
			return nil, fmt.Errorf("user is already a member of this organization")
		}
	}

	// Check if there's already a pending invitation
	pendingInvite, err := s.orgRepo.GetPendingInviteByEmail(orgID, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending invitation: %w", err)
	}
	if pendingInvite != nil {
		return nil, fmt.Errorf("invitation already pending for this email")
	}

	// Create invitation
	invite := &models.OrganizationInvite{
		OrganizationID: orgID,
		Email:          req.Email,
		Role:           req.Role,
		InvitedBy:      inviterID,
		Status:         models.InviteStatusPending,
	}

	if err := s.orgRepo.CreateInvite(invite); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// Get the full invitation with related data for the email
	fullInvite, err := s.orgRepo.GetInviteByID(invite.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get full invitation: %w", err)
	}

	// Send email invitation
	baseURL := "http://localhost:4200" // TODO: Get from config
	if err := s.emailService.SendOrganizationInvite(fullInvite, baseURL); err != nil {
		// Don't fail the entire operation if email fails, just log it
		fmt.Printf("Failed to send invitation email: %v\n", err)
	}

	return fullInvite, nil
}

// GetOrganizationInvites retrieves invitations for an organization (only admin can view)
func (s *OrganizationService) GetOrganizationInvites(orgID uuid.UUID, userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationInvite, int64, error) {
	// Get organization
	org, err := s.orgRepo.GetByID(orgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, 0, fmt.Errorf("organization not found")
	}

	// Check if user is admin
	if !s.isUserAdmin(org, userID) {
		return nil, 0, fmt.Errorf("insufficient permissions: only admins can view invitations")
	}

	return s.orgRepo.GetInvites(orgID, limit, offset, sortBy, sortOrder)
}

// AcceptInvite accepts an organization invitation
func (s *OrganizationService) AcceptInvite(token string, userID uuid.UUID) (*models.OrganizationMember, error) {
	// Get invitation by token
	invite, err := s.orgRepo.GetInviteByToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}
	if invite == nil {
		return nil, fmt.Errorf("invitation not found")
	}

	// Check if invitation is still valid
	if invite.Status != models.InviteStatusPending {
		return nil, fmt.Errorf("invitation is no longer valid")
	}
	if invite.IsExpired() {
		return nil, fmt.Errorf("invitation has expired")
	}

	// Get user to validate they exist and get their email
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if the invitation email matches the user's email
	if user.Username != invite.Email {
		return nil, fmt.Errorf("invitation email does not match user email")
	}

	// Check if user is already a member (including soft deleted)
	existingMember, err := s.orgRepo.GetMember(invite.OrganizationID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing membership: %w", err)
	}

	var member *models.OrganizationMember
	if existingMember != nil {
		// If member exists but is not soft deleted, they're already active
		if existingMember.DeletedAt.Time.IsZero() {
			return nil, fmt.Errorf("user is already a member of this organization")
		}

		// If member was soft deleted, restore them with new role
		existingMember.Role = invite.Role
		existingMember.DeletedAt = gorm.DeletedAt{}
		existingMember.JoinedAt = time.Now()
		if err := s.orgRepo.UpdateMember(existingMember); err != nil {
			return nil, fmt.Errorf("failed to restore member: %w", err)
		}
	} else {
		// Create new membership
		member = &models.OrganizationMember{
			OrganizationID: invite.OrganizationID,
			UserID:         userID,
			Role:           invite.Role,
		}
		if err := s.orgRepo.AddMember(member); err != nil {
			return nil, fmt.Errorf("failed to add member: %w", err)
		}
	}

	// Update invitation status
	invite.Status = models.InviteStatusAccepted
	now := time.Now()
	invite.AcceptedAt = &now
	if err := s.orgRepo.UpdateInvite(invite); err != nil {
		return nil, fmt.Errorf("failed to update invitation status: %w", err)
	}

	// Notify organization admins about the new member
	if err := s.notificationService.NotifyMemberJoined(invite.OrganizationID, user.Name, userID); err != nil {
		// Log the error but don't fail the invitation acceptance
		// Notification failure should not block the main operation
		fmt.Printf("Warning: Failed to notify admins about new member: %v\n", err)
	}

	return s.orgRepo.GetMember(invite.OrganizationID, userID)
}

// ValidateInvite validates an organization invitation
func (s *OrganizationService) ValidateInvite(token string) (map[string]interface{}, error) {
	// Get invitation by token
	invite, err := s.orgRepo.GetInviteByToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}
	if invite == nil {
		return nil, fmt.Errorf("invitation not found")
	}

	// Check if invitation is still valid
	if invite.Status != models.InviteStatusPending {
		return nil, fmt.Errorf("invitation is no longer valid")
	}
	if invite.IsExpired() {
		return nil, fmt.Errorf("invitation has expired")
	}

	// Check if user exists
	user, err := s.userRepo.GetByUsername(invite.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	result := map[string]interface{}{
		"valid":      true,
		"email":      invite.Email,
		"userExists": user != nil,
		"organization": map[string]interface{}{
			"id":   invite.OrganizationID,
			"name": invite.Organization.Name,
		},
		"role": invite.Role,
	}

	return result, nil
}

// RevokeInvite revokes an organization invitation (only admin can revoke)
func (s *OrganizationService) RevokeInvite(inviteID uuid.UUID, userID uuid.UUID) error {
	// Get invitation
	invite, err := s.orgRepo.GetInviteByID(inviteID)
	if err != nil {
		return fmt.Errorf("failed to get invitation: %w", err)
	}
	if invite == nil {
		return fmt.Errorf("invitation not found")
	}

	// Get organization
	org, err := s.orgRepo.GetByID(invite.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return fmt.Errorf("organization not found")
	}

	// Check if user is admin
	if !s.isUserAdmin(org, userID) {
		return fmt.Errorf("insufficient permissions: only admins can revoke invitations")
	}

	// Mark invitation as revoked
	invite.Status = models.InviteStatusRevoked
	return s.orgRepo.UpdateInvite(invite)
}

// ResendInvite resends an organization invitation with a new token and extended expiry
func (s *OrganizationService) ResendInvite(inviteID uuid.UUID, userID uuid.UUID) (*models.OrganizationInvite, error) {
	// Get invitation
	invite, err := s.orgRepo.GetInviteByID(inviteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}
	if invite == nil {
		return nil, fmt.Errorf("invitation not found")
	}

	// Get organization
	org, err := s.orgRepo.GetByID(invite.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, fmt.Errorf("organization not found")
	}

	// Check if user is admin
	if !s.isUserAdmin(org, userID) {
		return nil, fmt.Errorf("insufficient permissions: only admins can resend invitations")
	}

	// Check if invitation is pending (can only resend pending invitations)
	if invite.Status != models.InviteStatusPending {
		return nil, fmt.Errorf("can only resend pending invitations")
	}

	// Generate new token and extend expiry using repository method
	if err := s.orgRepo.RegenerateInviteToken(invite); err != nil {
		return nil, fmt.Errorf("failed to regenerate invitation token: %w", err)
	}

	// Get the updated invitation with related data for the email
	fullInvite, err := s.orgRepo.GetInviteByID(invite.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get full invitation: %w", err)
	}

	// Send email invitation
	baseURL := "http://localhost:4200" // TODO: Get from config
	if err := s.emailService.SendOrganizationInvite(fullInvite, baseURL); err != nil {
		// Don't fail the entire operation if email fails, just log it
		fmt.Printf("Failed to resend invitation email: %v\n", err)
	}

	return fullInvite, nil
}

// Helper Methods

// isUserAdmin checks if a user has administrative privileges in an organization.
// This method determines admin status by checking two conditions:
// 1. If the user is the original creator of the organization (always admin)
// 2. If the user is a member with admin role assigned
//
// Parameters:
//   - org: The organization to check admin status for
//   - userID: UUID of the user to check
//
// Returns:
//   - bool: true if user has admin privileges, false otherwise
//
// Business Rules:
//   - Organization creators are always considered admins regardless of member role
//   - Members with OrganizationRoleAdmin are considered admins
//   - Non-members and members with OrganizationRoleUser are not admins
//   - If database errors occur during member lookup, user is treated as non-admin
func (s *OrganizationService) isUserAdmin(org *models.Organization, userID uuid.UUID) bool {
	// Creator is always admin - this cannot be changed
	if org.CreatedBy == userID {
		return true
	}

	// Check membership role for non-creators
	member, err := s.orgRepo.GetMember(org.ID, userID)
	if err != nil || member == nil {
		// If error or not a member, user is not admin
		return false
	}

	// Return true only if member has admin role
	return member.Role == models.OrganizationRoleAdmin
}
