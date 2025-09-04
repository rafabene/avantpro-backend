package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	problemErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// NotificationController handles HTTP requests for notification operations
type NotificationController struct {
	notificationService services.NotificationService
}

// NewNotificationController creates a new NotificationController instance
func NewNotificationController(notificationService services.NotificationService) *NotificationController {
	return &NotificationController{
		notificationService: notificationService,
	}
}

// getOrganizationIDFromHeader extracts and validates the Organization-ID header
func (c *NotificationController) getOrganizationIDFromHeader(ctx *gin.Context) (*uuid.UUID, error) {
	orgIDHeader := ctx.GetHeader("Organization-ID")
	if orgIDHeader == "" {
		return nil, errors.New("Organization-ID header is required")
	}

	orgID, err := uuid.Parse(orgIDHeader)
	if err != nil {
		return nil, errors.New("invalid Organization-ID format")
	}

	return &orgID, nil
}

// GetUserNotifications retrieves paginated notifications for the authenticated user
// @Summary Get user notifications
// @Description Retrieve paginated list of notifications for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param page query int false "Page number (default: 1)" default(1)
// @Param limit query int false "Number of items per page (default: 10, max: 100)" default(10)
// @Param sortBy query string false "Sort by field (title, type, read, created_at, updated_at)" default(created_at)
// @Param sortOrder query string false "Sort order (asc, desc)" default(desc)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Success response with notifications and pagination"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notifications [get]
func (c *NotificationController) GetUserNotifications(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get organization ID from header
	organizationID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Parse pagination parameters
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	sortBy := ctx.DefaultQuery("sortBy", "created_at")
	sortOrder := ctx.DefaultQuery("sortOrder", "desc")

	// Calculate offset
	offset := (page - 1) * limit

	// Get notifications
	notifications, total, err := c.notificationService.GetUserNotifications(
		userID.(uuid.UUID),
		organizationID,
		limit,
		offset,
		sortBy,
		sortOrder,
	)
	if err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Calculate pagination info
	totalPages := (int(total) + limit - 1) / limit

	ctx.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetUnreadNotifications retrieves all unread notifications for the authenticated user
// @Summary Get unread notifications
// @Description Retrieve all unread notifications for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Success response with unread notifications"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notifications/unread [get]
func (c *NotificationController) GetUnreadNotifications(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get organization ID from header
	organizationID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get unread notifications
	notifications, err := c.notificationService.GetUnreadNotifications(userID.(uuid.UUID), organizationID)
	if err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

// GetUnreadCount retrieves the count of unread notifications for the authenticated user
// @Summary Get unread notifications count
// @Description Retrieve the count of unread notifications for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Success response with unread count"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notifications/unread-count [get]
func (c *NotificationController) GetUnreadCount(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get organization ID from header
	organizationID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Get unread count
	count, err := c.notificationService.GetUnreadCount(userID.(uuid.UUID), organizationID)
	if err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// MarkAsRead marks a specific notification as read
// @Summary Mark notification as read
// @Description Mark a specific notification as read for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param notifId path string true "Notification ID"
// @Security BearerAuth
// @Success 200 {object} models.MessageResponse "Success message"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Notification not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notifications/{notifId}/read [put]
func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Parse notification ID
	notificationIDStr := ctx.Param("notifId")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		prob := problemErrors.BadRequestError("Invalid notification ID: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Mark as read
	if err := c.notificationService.MarkAsRead(notificationID, userID.(uuid.UUID)); err != nil {
		if err.Error() == "notification not found" {
			prob := problemErrors.NotFoundError("Notification not found", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		if err.Error() == "notification does not belong to user" {
			prob := problemErrors.ForbiddenError("Access denied", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, models.MessageResponse{
		Message: "Notification marked as read successfully",
	})
}

// MarkAllAsRead marks all notifications as read for the authenticated user
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Security BearerAuth
// @Success 200 {object} models.MessageResponse "Success message"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notifications/mark-all-read [put]
func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Mark all as read
	if err := c.notificationService.MarkAllAsRead(userID.(uuid.UUID)); err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, models.MessageResponse{
		Message: "All notifications marked as read successfully",
	})
}

// DeleteNotification deletes a specific notification
// @Summary Delete notification
// @Description Delete a specific notification for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Param notifId path string true "Notification ID"
// @Security BearerAuth
// @Success 200 {object} models.MessageResponse "Success message"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Notification not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notifications/{notifId} [delete]
func (c *NotificationController) DeleteNotification(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Parse notification ID
	notificationIDStr := ctx.Param("notifId")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		prob := problemErrors.BadRequestError("Invalid notification ID: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Delete notification
	if err := c.notificationService.DeleteNotification(notificationID, userID.(uuid.UUID)); err != nil {
		if err.Error() == "notification not found" {
			prob := problemErrors.NotFoundError("Notification not found", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		if err.Error() == "notification does not belong to user" {
			prob := problemErrors.ForbiddenError("Access denied", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, models.MessageResponse{
		Message: "Notification deleted successfully",
	})
}

// DeleteAllNotifications deletes all notifications for the authenticated user
// @Summary Delete all notifications
// @Description Delete all notifications for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "Organization ID"
// @Security BearerAuth
// @Success 200 {object} models.MessageResponse "Success message"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /organizations/notifications [delete]
func (c *NotificationController) DeleteAllNotifications(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("userID")
	if !exists {
		prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Delete all notifications
	if err := c.notificationService.DeleteAllNotifications(userID.(uuid.UUID)); err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, models.MessageResponse{
		Message: "All notifications deleted successfully",
	})
}
