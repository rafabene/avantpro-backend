package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	problemErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/middleware"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// OrganizationController handles organization-related HTTP requests
type OrganizationController struct {
	service services.OrganizationServiceInterface
}

// NewOrganizationController creates a new organization controller
func NewOrganizationController(service services.OrganizationServiceInterface) *OrganizationController {
	return &OrganizationController{
		service: service,
	}
}

// getOrganizationIDFromHeader extracts and validates the Organization-ID header
func (c *OrganizationController) getOrganizationIDFromHeader(ctx *gin.Context) (uuid.UUID, error) {
	orgIDHeader := ctx.GetHeader("Organization-ID")
	if orgIDHeader == "" {
		return uuid.Nil, errors.New("Organization-ID header is required")
	}

	orgID, err := uuid.Parse(orgIDHeader)
	if err != nil {
		return uuid.Nil, errors.New("invalid Organization-ID format")
	}

	return orgID, nil
}

// CreateOrganization creates a new organization
// @Summary Create a new organization
// @Description Create a new organization with the authenticated user as admin
// @Tags organizations
// @Accept json
// @Produce json
// @Param organization body models.OrganizationCreateRequest true "Organization data"
// @Success 201 {object} models.OrganizationResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations [post]
func (c *OrganizationController) CreateOrganization(ctx *gin.Context) {
	// Get user ID from JWT token
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	var req models.OrganizationCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	org, err := c.service.CreateOrganization(&req, userID)
	if err != nil {
		problemErrors.HandleInternalError(ctx, "Failed to create organization", err)
		return
	}

	response := c.convertToOrganizationResponse(org)
	ctx.JSON(http.StatusCreated, response)
}

// GetOrganization retrieves an organization by ID
// @Summary Get organization by ID
// @Description Get a specific organization by its ID
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Success 200 {object} models.OrganizationResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations [get]
func (c *OrganizationController) GetOrganization(ctx *gin.Context) {
	id, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	org, err := c.service.GetOrganization(id)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organization not found")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to get organization", err)
		return
	}

	response := c.convertToOrganizationResponse(org)
	ctx.JSON(http.StatusOK, response)
}

// GetUserOrganizations retrieves organizations created by the authenticated user
// @Summary Get user's organizations
// @Description Get all organizations created by the authenticated user
// @Tags organizations
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Param sortBy query string false "Sort by field" default("created_at")
// @Param sortOrder query string false "Sort order (asc/desc)" default("desc")
// @Success 200 {object} models.OrganizationListResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/my [get]
func (c *OrganizationController) GetUserOrganizations(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	// Parse pagination parameters
	page, limit, sortBy, sortOrder := c.parsePaginationParams(ctx)
	offset := (page - 1) * limit

	orgs, total, err := c.service.GetUserOrganizations(userID, limit, offset, sortBy, sortOrder)
	if err != nil {
		problemErrors.HandleInternalError(ctx, "Failed to get user organizations", err)
		return
	}

	response := models.OrganizationListResponse{
		Data:  c.convertToOrganizationResponseList(orgs),
		Total: total,
		Limit: limit,
		Page:  page,
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateOrganization updates an organization
// @Summary Update organization
// @Description Update an organization (admin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param organization body models.OrganizationUpdateRequest true "Update data"
// @Success 200 {object} models.OrganizationResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations [put]
func (c *OrganizationController) UpdateOrganization(ctx *gin.Context) {
	id, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	var req models.OrganizationUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	org, err := c.service.UpdateOrganization(id, &req, userID)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organization not found")
			return
		}
		if err.Error() == "insufficient permissions: only admins can update organization" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to update organization", err)
		return
	}

	response := c.convertToOrganizationResponse(org)
	ctx.JSON(http.StatusOK, response)
}

// DeleteOrganization deletes an organization
// @Summary Delete organization
// @Description Delete an organization (creator only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Success 204
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations [delete]
func (c *OrganizationController) DeleteOrganization(ctx *gin.Context) {
	id, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	err = c.service.DeleteOrganization(id, userID)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organization not found")
			return
		}
		if err.Error() == "insufficient permissions: only the creator can delete organization" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to delete organization", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetOrganizationMembers retrieves members of an organization
// @Summary Get organization members
// @Description Get all members of an organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Param sortBy query string false "Sort by field" default("joined_at")
// @Param sortOrder query string false "Sort order (asc/desc)" default("desc")
// @Success 200 {object} models.OrganizationMemberListResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/members [get]
func (c *OrganizationController) GetOrganizationMembers(ctx *gin.Context) {
	idStr := ctx.Param("id")
	orgID, err := uuid.Parse(idStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Invalid organization ID format")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	// Parse pagination parameters
	page, limit, sortBy, sortOrder := c.parsePaginationParams(ctx)
	offset := (page - 1) * limit

	members, total, err := c.service.GetOrganizationMembers(orgID, userID, limit, offset, sortBy, sortOrder)
	if err != nil {
		if err.Error() == "insufficient permissions: user is not a member of this organization" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to get organization members", err)
		return
	}

	response := models.OrganizationMemberListResponse{
		Data:  c.convertToOrganizationMemberResponseList(members),
		Total: total,
		Limit: limit,
		Page:  page,
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateMemberRole updates a member's role
// @Summary Update member role
// @Description Update a member's role in the organization (admin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param userId path string true "User ID"
// @Param member body models.OrganizationMemberUpdateRequest true "Role update data"
// @Success 200 {object} models.OrganizationMemberResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/members/{userId} [put]
func (c *OrganizationController) UpdateMemberRole(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	memberUserIDStr := ctx.Param("userId")
	memberUserID, err := uuid.Parse(memberUserIDStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Invalid user ID format")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	var req models.OrganizationMemberUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	member, err := c.service.UpdateMemberRole(orgID, memberUserID, &req, userID)
	if err != nil {
		if err.Error() == "organization not found" || err.Error() == "member not found" {
			problemErrors.HandleNotFoundError(ctx, "Resource not found")
			return
		}
		if err.Error() == "insufficient permissions: only admins can update member roles" ||
			err.Error() == "cannot change role of organization creator" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to update member role", err)
		return
	}

	response := c.convertToOrganizationMemberResponse(member)
	ctx.JSON(http.StatusOK, response)
}

// RemoveMember removes a member from an organization
// @Summary Remove member
// @Description Remove a member from the organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param userId path string true "User ID"
// @Success 204
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/members/{userId} [delete]
func (c *OrganizationController) RemoveMember(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	memberUserIDStr := ctx.Param("userId")
	memberUserID, err := uuid.Parse(memberUserIDStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Invalid user ID format")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	err = c.service.RemoveMember(orgID, memberUserID, userID)
	if err != nil {
		if err.Error() == "organization not found" || err.Error() == "member not found" {
			problemErrors.HandleNotFoundError(ctx, "Resource not found")
			return
		}
		if err.Error() == "cannot remove organization creator" ||
			err.Error() == "insufficient permissions: only admins can remove other members" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to remove member", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// InviteUser sends an invitation to join an organization
// @Summary Invite user to organization
// @Description Send an invitation to a user to join the organization (admin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param invitation body models.OrganizationInviteRequest true "Invitation data"
// @Success 201 {object} models.OrganizationInviteResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/invites [post]
func (c *OrganizationController) InviteUser(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	var req models.OrganizationInviteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	invite, err := c.service.InviteUser(orgID, &req, userID)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organization not found")
			return
		}
		if err.Error() == "insufficient permissions: only admins can send invitations" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		if err.Error() == "user is already a member of this organization" ||
			err.Error() == "invitation already pending for this email" {
			problemErrors.HandleConflictError(ctx, err.Error())
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to send invitation", err)
		return
	}

	response := c.convertToOrganizationInviteResponse(invite)
	ctx.JSON(http.StatusCreated, response)
}

// GetOrganizationInvites retrieves invitations for an organization
// @Summary Get organization invitations
// @Description Get all pending invitations for the organization (admin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Param sortBy query string false "Sort by field" default("created_at")
// @Param sortOrder query string false "Sort order (asc/desc)" default("desc")
// @Success 200 {object} models.OrganizationInviteListResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/invites [get]
func (c *OrganizationController) GetOrganizationInvites(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	// Parse pagination parameters
	page, limit, sortBy, sortOrder := c.parsePaginationParams(ctx)
	offset := (page - 1) * limit

	invites, total, err := c.service.GetOrganizationInvites(orgID, userID, limit, offset, sortBy, sortOrder)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organization not found")
			return
		}
		if err.Error() == "insufficient permissions: only admins can view invitations" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to get organization invitations", err)
		return
	}

	response := models.OrganizationInviteListResponse{
		Data:  c.convertToOrganizationInviteResponseList(invites),
		Total: total,
		Limit: limit,
		Page:  page,
	}

	ctx.JSON(http.StatusOK, response)
}

// AcceptInvite accepts an organization invitation
// @Summary Accept organization invitation
// @Description Accept an invitation to join an organization using the invitation token
// @Tags organizations
// @Accept json
// @Produce json
// @Param token path string true "Invitation token"
// @Success 200 {object} models.OrganizationMemberResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 410 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/invites/{token}/accept [post]
func (c *OrganizationController) AcceptInvite(ctx *gin.Context) {
	token := ctx.Param("token")
	if token == "" {
		problemErrors.HandleValidationError(ctx, "Invitation token required")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	member, err := c.service.AcceptInvite(token, userID)
	if err != nil {
		if err.Error() == "invitation not found" {
			problemErrors.HandleNotFoundError(ctx, "Invitation not found")
			return
		}
		if err.Error() == "invitation is no longer valid" || err.Error() == "invitation has expired" {
			problemErrors.HandleGoneError(ctx, err.Error())
			return
		}
		if err.Error() == "user not found" {
			problemErrors.HandleNotFoundError(ctx, "User not found")
			return
		}
		if err.Error() == "invitation email does not match user email" ||
			err.Error() == "user is already a member of this organization" {
			problemErrors.HandleConflictError(ctx, err.Error())
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to accept invitation", err)
		return
	}

	response := c.convertToOrganizationMemberResponse(member)
	ctx.JSON(http.StatusOK, response)
}

// ValidateInvite validates an organization invitation token without requiring authentication
// @Summary Validate organization invitation
// @Description Validate an invitation token and check if the invited user exists
// @Tags organizations
// @Produce json
// @Param token path string true "Invitation Token"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} errors.ProblemDetail
// @Failure 410 {object} errors.ProblemDetail
// @Router /organizations/invites/token/{token}/validate [get]
func (c *OrganizationController) ValidateInvite(ctx *gin.Context) {
	token := ctx.Param("token")
	if token == "" {
		problemErrors.HandleValidationError(ctx, "Invitation token required")
		return
	}

	result, err := c.service.ValidateInvite(token)
	if err != nil {
		if err.Error() == "invitation not found" {
			problemErrors.HandleNotFoundError(ctx, "Invitation not found")
			return
		}
		if err.Error() == "invitation is no longer valid" || err.Error() == "invitation has expired" {
			problemErrors.HandleGoneError(ctx, err.Error())
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to validate invitation", err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// RevokeInvite revokes an organization invitation
// @Summary Revoke organization invitation
// @Description Revoke an invitation to join the organization (admin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param inviteId path string true "Invitation ID"
// @Success 204
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/invites/{inviteId} [delete]
func (c *OrganizationController) RevokeInvite(ctx *gin.Context) {
	inviteIDStr := ctx.Param("inviteId")
	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Invalid invitation ID format")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	err = c.service.RevokeInvite(inviteID, userID)
	if err != nil {
		if err.Error() == "invitation not found" || err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Resource not found")
			return
		}
		if err.Error() == "insufficient permissions: only admins can revoke invitations" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to revoke invitation", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetUserMemberships retrieves all organizations the user is a member of
// @Summary Get user memberships
// @Description Get all organizations the authenticated user is a member of
// @Tags organizations
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Param sortBy query string false "Sort by field" default("joined_at")
// @Param sortOrder query string false "Sort order (asc/desc)" default("desc")
// @Success 200 {object} models.OrganizationMemberListResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/memberships [get]
func (c *OrganizationController) GetUserMemberships(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	// Parse pagination parameters
	page, limit, sortBy, sortOrder := c.parsePaginationParams(ctx)
	offset := (page - 1) * limit

	memberships, total, err := c.service.GetUserMemberships(userID, limit, offset, sortBy, sortOrder)
	if err != nil {
		problemErrors.HandleInternalError(ctx, "Failed to get user memberships", err)
		return
	}

	response := models.OrganizationMemberListResponse{
		Data:  c.convertToOrganizationMemberResponseList(memberships),
		Total: total,
		Limit: limit,
		Page:  page,
	}

	ctx.JSON(http.StatusOK, response)
}

// Helper Methods

// parsePaginationParams parses pagination parameters from query string
func (c *OrganizationController) parsePaginationParams(ctx *gin.Context) (page, limit int, sortBy, sortOrder string) {
	page = 1
	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit = 50
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	sortBy = ctx.DefaultQuery("sortBy", "created_at")
	sortOrder = ctx.DefaultQuery("sortOrder", "desc")

	return
}

// Conversion Methods

func (c *OrganizationController) convertToOrganizationResponse(org *models.Organization) *models.OrganizationResponse {
	response := &models.OrganizationResponse{
		ID:          org.ID,
		Name:        org.Name,
		Description: org.Description,
		CreatedBy:   org.CreatedBy,
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
	}

	if org.Creator.ID != uuid.Nil {
		response.Creator = &models.UserResponse{
			ID:       org.Creator.ID,
			Username: org.Creator.Username,
			Name:     org.Creator.Name,
		}
	}

	if len(org.Members) > 0 {
		response.Members = c.convertToOrganizationMemberResponseList(org.Members)
	}

	if len(org.Invites) > 0 {
		response.Invites = c.convertToOrganizationInviteResponseList(org.Invites)
	}

	return response
}

func (c *OrganizationController) convertToOrganizationResponseList(orgs []models.Organization) []models.OrganizationResponse {
	responses := make([]models.OrganizationResponse, len(orgs))
	for i, org := range orgs {
		responses[i] = *c.convertToOrganizationResponse(&org)
	}
	return responses
}

func (c *OrganizationController) convertToOrganizationMemberResponse(member *models.OrganizationMember) *models.OrganizationMemberResponse {
	response := &models.OrganizationMemberResponse{
		ID:             member.ID,
		OrganizationID: member.OrganizationID,
		UserID:         member.UserID,
		Role:           member.Role,
		JoinedAt:       member.JoinedAt,
		CreatedAt:      member.CreatedAt,
		UpdatedAt:      member.UpdatedAt,
	}

	if member.User.ID != uuid.Nil {
		response.User = &models.UserResponse{
			ID:       member.User.ID,
			Username: member.User.Username,
			Name:     member.User.Name,
		}
	}

	if member.Organization.ID != uuid.Nil {
		response.Organization = &models.OrganizationResponse{
			ID:          member.Organization.ID,
			Name:        member.Organization.Name,
			Description: member.Organization.Description,
			CreatedBy:   member.Organization.CreatedBy,
			CreatedAt:   member.Organization.CreatedAt,
			UpdatedAt:   member.Organization.UpdatedAt,
		}
	}

	return response
}

func (c *OrganizationController) convertToOrganizationMemberResponseList(members []models.OrganizationMember) []models.OrganizationMemberResponse {
	responses := make([]models.OrganizationMemberResponse, len(members))
	for i, member := range members {
		responses[i] = *c.convertToOrganizationMemberResponse(&member)
	}
	return responses
}

func (c *OrganizationController) convertToOrganizationInviteResponse(invite *models.OrganizationInvite) *models.OrganizationInviteResponse {
	response := &models.OrganizationInviteResponse{
		ID:             invite.ID,
		OrganizationID: invite.OrganizationID,
		Email:          invite.Email,
		Role:           invite.Role,
		InvitedBy:      invite.InvitedBy,
		Status:         invite.Status,
		ExpiresAt:      invite.ExpiresAt,
		AcceptedAt:     invite.AcceptedAt,
		CreatedAt:      invite.CreatedAt,
		UpdatedAt:      invite.UpdatedAt,
	}

	if invite.Organization.ID != uuid.Nil {
		response.Organization = &models.OrganizationResponse{
			ID:          invite.Organization.ID,
			Name:        invite.Organization.Name,
			Description: invite.Organization.Description,
			CreatedBy:   invite.Organization.CreatedBy,
			CreatedAt:   invite.Organization.CreatedAt,
			UpdatedAt:   invite.Organization.UpdatedAt,
		}
	}

	if invite.Inviter.ID != uuid.Nil {
		response.Inviter = &models.UserResponse{
			ID:       invite.Inviter.ID,
			Username: invite.Inviter.Username,
			Name:     invite.Inviter.Name,
		}
	}

	return response
}

func (c *OrganizationController) convertToOrganizationInviteResponseList(invites []models.OrganizationInvite) []models.OrganizationInviteResponse {
	responses := make([]models.OrganizationInviteResponse, len(invites))
	for i, invite := range invites {
		responses[i] = *c.convertToOrganizationInviteResponse(&invite)
	}
	return responses
}

// ResendInvite resends an organization invitation
// @Summary Resend organization invitation
// @Description Resend an invitation with a new token and extended expiry (admin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Param inviteId path string true "Invitation ID"
// @Success 200 {object} models.OrganizationInviteResponse
// @Failure 400 {object} errors.ProblemDetail
// @Failure 401 {object} errors.ProblemDetail
// @Failure 403 {object} errors.ProblemDetail
// @Failure 404 {object} errors.ProblemDetail
// @Failure 500 {object} errors.ProblemDetail
// @Router /organizations/invites/{inviteId}/resend [post]
func (c *OrganizationController) ResendInvite(ctx *gin.Context) {
	inviteIDStr := ctx.Param("inviteId")
	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Invalid invitation ID format")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		problemErrors.HandleUnauthorizedError(ctx, "User not authenticated")
		return
	}

	invite, err := c.service.ResendInvite(inviteID, userID)
	if err != nil {
		if err.Error() == "invitation not found" || err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Resource not found")
			return
		}
		if err.Error() == "insufficient permissions: only admins can resend invitations" {
			problemErrors.HandleForbiddenError(ctx, "Insufficient permissions")
			return
		}
		if err.Error() == "can only resend pending invitations" {
			problemErrors.HandleValidationError(ctx, "Can only resend pending invitations")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to resend invitation", err)
		return
	}

	response := c.convertToOrganizationInviteResponse(invite)
	ctx.JSON(http.StatusOK, response)
}
