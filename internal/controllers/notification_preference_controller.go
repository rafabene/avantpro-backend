package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	problemErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// NotificationPreferenceController handles HTTP requests for notification preference operations
type NotificationPreferenceController struct {
	preferenceService services.NotificationPreferenceService
}

// NewNotificationPreferenceController creates a new NotificationPreferenceController instance
func NewNotificationPreferenceController(preferenceService services.NotificationPreferenceService) *NotificationPreferenceController {
	return &NotificationPreferenceController{
		preferenceService: preferenceService,
	}
}

// getOrganizationIDFromHeader extracts and validates the Organization-ID header
func (c *NotificationPreferenceController) getOrganizationIDFromHeader(ctx *gin.Context) (uuid.UUID, error) {
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

// GetOrganizationPreferences retrieves notification preferences for an organization
// @Summary Get organization notification preferences
// @Description Retrieve notification preferences for an organization. Creates defaults if none exist.
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Organization-ID header string true "Organization ID"
// @Success 200 {object} map[string]interface{} "Success response with preferences"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Organization not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notification-preferences [get]
func (c *NotificationPreferenceController) GetOrganizationPreferences(ctx *gin.Context) {
	// Get user ID from JWT middleware (for authentication)
	_, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get organization ID from header
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get preferences
	preferences, err := c.preferenceService.GetOrganizationPreferences(organizationUUID)
	if err != nil {
		if err.Error() == "organization not found" {
			prob := problemErrors.NotFoundError("Organization not found", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"preferences": preferences,
	})
}

// UpdateOrganizationPreferences updates notification preferences for an organization
// @Summary Update organization notification preferences
// @Description Bulk update notification preferences for an organization
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param request body models.NotificationPreferenceBulkUpdateRequest true "Bulk update request"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Success response with updated preferences"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Organization not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notification-preferences [put]
func (c *NotificationPreferenceController) UpdateOrganizationPreferences(ctx *gin.Context) {
	// Get user ID from JWT middleware (for authentication)
	_, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get organization ID from header
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Parse request body
	var req models.NotificationPreferenceBulkUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Invalid JSON format: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Update preferences
	preferences, err := c.preferenceService.UpdateOrganizationPreferences(organizationUUID, &req)
	if err != nil {
		if err.Error() == "organization not found" {
			prob := problemErrors.NotFoundError("Organization not found", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.BadRequestError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"preferences": preferences,
		"message":     "Preferences updated successfully",
	})
}

// UpdateSinglePreference updates a single notification preference
// @Summary Update single notification preference
// @Description Update a specific notification preference for an organization
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param event path string true "Notification event type"
// @Param request body models.NotificationPreferenceUpdateRequest true "Update request"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Success response with updated preference"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Organization not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notification-preferences/{event} [put]
func (c *NotificationPreferenceController) UpdateSinglePreference(ctx *gin.Context) {
	// Get user ID from JWT middleware (for authentication)
	_, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get organization ID from header
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Parse event parameter
	eventStr := ctx.Param("event")
	event := models.NotificationEvent(eventStr)

	// Parse request body
	var req models.NotificationPreferenceUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Invalid JSON format: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Update single preference
	preference, err := c.preferenceService.UpdateSinglePreference(organizationUUID, event, &req)
	if err != nil {
		if err.Error() == "organization not found" {
			prob := problemErrors.NotFoundError("Organization not found", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.BadRequestError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"preference": preference,
		"message":    "Preference updated successfully",
	})
}

// ResetToDefaults resets organization preferences to default values
// @Summary Reset preferences to defaults
// @Description Reset all notification preferences for an organization to default values
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Success response with default preferences"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Organization not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notification-preferences/reset [post]
func (c *NotificationPreferenceController) ResetToDefaults(ctx *gin.Context) {
	// Get user ID from JWT middleware (for authentication)
	_, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get organization ID from header
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Reset to defaults
	preferences, err := c.preferenceService.ResetToDefaults(organizationUUID)
	if err != nil {
		if err.Error() == "organization not found" {
			prob := problemErrors.NotFoundError("Organization not found", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"preferences": preferences,
		"message":     "Preferences reset to defaults successfully",
	})
}

// GetAvailableEvents returns all available notification events
// @Summary Get available notification events
// @Description Retrieve all available notification event types with descriptions
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Success response with available events"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /notification-preferences/events [get]
func (c *NotificationPreferenceController) GetAvailableEvents(ctx *gin.Context) {
	// Get user ID from JWT middleware (for authentication only)
	_, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get available events
	events := c.preferenceService.GetAvailableEvents()

	ctx.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// GenerateTestNotification creates a test notification
// @Summary Generate test notification
// @Description Generate a test notification for the authenticated user to verify settings
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Param request body models.TestNotificationRequest true "Test notification request"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{} "Success response with test notification"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /notification-preferences/test [post]
func (c *NotificationPreferenceController) GenerateTestNotification(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get organization ID from header
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Parse request body
	var req models.TestNotificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Invalid JSON format: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Set organization ID from path parameter
	req.OrganizationID = organizationUUID

	// Generate test notification
	notification, err := c.preferenceService.GenerateTestNotification(userID.(uuid.UUID), &req)
	if err != nil {
		if err.Error() == "user not found" {
			prob := problemErrors.NotFoundError("User not found", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"notification": notification,
		"message":      "Test notification created successfully",
	})
}
