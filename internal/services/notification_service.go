package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/repositories"
)

// NotificationService interface defines notification management operations.
// This interface provides methods for creating, retrieving, and managing user notifications.
type NotificationService interface {
	// CreateNotification creates a new notification for a user.
	// This method validates the notification data and creates it in the database.
	// Parameters:
	//   - req: Create notification request containing user ID, title, message, type, etc.
	// Returns:
	//   - *NotificationResponse: Created notification information
	//   - error: Error if validation fails or creation fails
	CreateNotification(req *CreateNotificationRequest) (*NotificationResponse, error)

	// NotifyMemberJoined creates a notification for organization admins when a new member joins.
	// This method finds all organization admins and creates notifications for them.
	// Parameters:
	//   - organizationID: ID of the organization
	//   - newMemberName: Name of the new member who joined
	//   - newMemberID: ID of the new member who joined
	// Returns:
	//   - error: Error if notification creation fails
	NotifyMemberJoined(organizationID uuid.UUID, newMemberName string, newMemberID uuid.UUID) error

	// GetUserNotifications retrieves paginated notifications for a specific user.
	// This method returns notifications with sorting and pagination support.
	// Parameters:
	//   - userID: ID of the user to get notifications for
	//   - organizationID: Optional organization ID to filter notifications (nil for all organizations)
	//   - limit: Maximum number of notifications to return
	//   - offset: Number of notifications to skip for pagination
	//   - sortBy: Field to sort by (title, type, read, created_at, updated_at)
	//   - sortOrder: Sort order (asc or desc)
	// Returns:
	//   - []NotificationResponse: List of notifications
	//   - int64: Total count of notifications for pagination
	//   - error: Error if retrieval fails
	GetUserNotifications(userID uuid.UUID, organizationID *uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]NotificationResponse, int64, error)

	// GetUnreadNotifications retrieves all unread notifications for a user.
	// This method returns notifications that have not been marked as read.
	// Parameters:
	//   - userID: ID of the user to get unread notifications for
	//   - organizationID: Optional organization ID to filter notifications (nil for all organizations)
	// Returns:
	//   - []NotificationResponse: List of unread notifications
	//   - error: Error if retrieval fails
	GetUnreadNotifications(userID uuid.UUID, organizationID *uuid.UUID) ([]NotificationResponse, error)

	// GetUnreadCount retrieves the count of unread notifications for a user.
	// This method returns the number of unread notifications for display in UI badges.
	// Parameters:
	//   - userID: ID of the user to count unread notifications for
	//   - organizationID: Optional organization ID to filter notifications (nil for all organizations)
	// Returns:
	//   - int64: Number of unread notifications
	//   - error: Error if count retrieval fails
	GetUnreadCount(userID uuid.UUID, organizationID *uuid.UUID) (int64, error)

	// MarkAsRead marks a specific notification as read.
	// This method updates the notification status and sets the read timestamp.
	// Parameters:
	//   - notificationID: ID of the notification to mark as read
	//   - userID: ID of the user (for authorization)
	// Returns:
	//   - error: Error if notification not found or user not authorized
	MarkAsRead(notificationID, userID uuid.UUID) error

	// MarkAllAsRead marks all notifications for a user as read.
	// This method updates all unread notifications for the user.
	// Parameters:
	//   - userID: ID of the user to mark all notifications as read
	// Returns:
	//   - error: Error if update fails
	MarkAllAsRead(userID uuid.UUID) error

	// DeleteNotification removes a notification for a user.
	// This method performs a soft delete of the notification.
	// Parameters:
	//   - notificationID: ID of the notification to delete
	//   - userID: ID of the user (for authorization)
	// Returns:
	//   - error: Error if notification not found or user not authorized
	DeleteNotification(notificationID, userID uuid.UUID) error

	// DeleteAllNotifications removes all notifications for a user.
	// This method performs a soft delete of all user notifications.
	// Parameters:
	//   - userID: ID of the user to delete all notifications for
	// Returns:
	//   - error: Error if deletion fails
	DeleteAllNotifications(userID uuid.UUID) error
}

// notificationService implements the NotificationService interface
type notificationService struct {
	notificationRepo repositories.NotificationRepository
	organizationRepo repositories.OrganizationRepositoryInterface
	userRepo         repositories.UserRepository
}

// NewNotificationService creates a new NotificationService instance
func NewNotificationService(
	notificationRepo repositories.NotificationRepository,
	organizationRepo repositories.OrganizationRepositoryInterface,
	userRepo repositories.UserRepository,
	_ interface{}, // Deprecated WebSocket parameter, kept for compatibility
) NotificationService {
	return &notificationService{
		notificationRepo: notificationRepo,
		organizationRepo: organizationRepo,
		userRepo:         userRepo,
	}
}

// CreateNotification creates a new notification for a user
func (s *notificationService) CreateNotification(req *CreateNotificationRequest) (*NotificationResponse, error) {
	// Validate user exists
	_, err := s.userRepo.GetByID(req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to validate user: %w", err)
	}

	// Validate organization exists
	_, err = s.organizationRepo.GetByID(req.OrganizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to validate organization: %w", err)
	}

	// Create notification
	notification := &models.Notification{
		UserID:         req.UserID,
		OrganizationID: req.OrganizationID,
		Title:          req.Title,
		Message:        req.Message,
		Type:           models.NotificationType(req.Type),
		Data:           req.Data,
		Read:           false,
	}

	if err := s.notificationRepo.Create(notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Convert to response
	response := &NotificationResponse{
		ID:             notification.ID,
		Title:          notification.Title,
		Message:        notification.Message,
		Type:           NotificationType(notification.Type),
		Read:           notification.Read,
		ReadAt:         notification.ReadAt,
		Data:           notification.Data,
		OrganizationID: notification.OrganizationID,
		CreatedAt:      notification.CreatedAt,
	}

	// Notification created successfully

	return response, nil
}

// NotifyMemberJoined creates notifications for organization admins when a new member joins
func (s *notificationService) NotifyMemberJoined(organizationID uuid.UUID, newMemberName string, newMemberID uuid.UUID) error {
	// Get organization with members
	organization, err := s.organizationRepo.GetByID(organizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	// Get organization members with role admin (get all members, we'll filter for admins)
	members, _, err := s.organizationRepo.GetMembers(organizationID, 100, 0, "role", "desc")
	if err != nil {
		return fmt.Errorf("failed to get organization members: %w", err)
	}

	// Create notifications for all admin members
	for _, member := range members {
		if member.Role == models.OrganizationRoleAdmin {
			// Skip the new member themselves
			if member.UserID == newMemberID {
				continue
			}

			notificationReq := &CreateNotificationRequest{
				UserID:         member.UserID,
				OrganizationID: organizationID,
				Title:          "Novo membro na organização",
				Message:        fmt.Sprintf("%s foi adicionado à organização %s", newMemberName, organization.Name),
				Type:           NotificationTypeInfo,
				Data:           fmt.Sprintf("{\"action\":\"member_joined\",\"member_id\":\"%s\",\"organization_id\":\"%s\"}", newMemberID, organizationID),
			}

			_, err := s.CreateNotification(notificationReq)
			if err != nil {
				// Log the error but don't fail the entire process
				continue
			}
		}
	}

	return nil
}

// GetUserNotifications retrieves paginated notifications for a specific user
func (s *notificationService) GetUserNotifications(userID uuid.UUID, organizationID *uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]NotificationResponse, int64, error) {
	notifications, total, err := s.notificationRepo.GetByUserID(userID, organizationID, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user notifications: %w", err)
	}

	// Convert to response format
	responses := make([]NotificationResponse, len(notifications))
	for i, notification := range notifications {
		responses[i] = NotificationResponse{
			ID:             notification.ID,
			Title:          notification.Title,
			Message:        notification.Message,
			Type:           NotificationType(notification.Type),
			Read:           notification.Read,
			ReadAt:         notification.ReadAt,
			Data:           notification.Data,
			OrganizationID: notification.OrganizationID,
			CreatedAt:      notification.CreatedAt,
		}
	}

	return responses, total, nil
}

// GetUnreadNotifications retrieves all unread notifications for a user
func (s *notificationService) GetUnreadNotifications(userID uuid.UUID, organizationID *uuid.UUID) ([]NotificationResponse, error) {
	notifications, err := s.notificationRepo.GetUnreadByUserID(userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread notifications: %w", err)
	}

	// Convert to response format
	responses := make([]NotificationResponse, len(notifications))
	for i, notification := range notifications {
		responses[i] = NotificationResponse{
			ID:             notification.ID,
			Title:          notification.Title,
			Message:        notification.Message,
			Type:           NotificationType(notification.Type),
			Read:           notification.Read,
			ReadAt:         notification.ReadAt,
			Data:           notification.Data,
			OrganizationID: notification.OrganizationID,
			CreatedAt:      notification.CreatedAt,
		}
	}

	return responses, nil
}

// GetUnreadCount retrieves the count of unread notifications for a user
func (s *notificationService) GetUnreadCount(userID uuid.UUID, organizationID *uuid.UUID) (int64, error) {
	count, err := s.notificationRepo.GetUnreadCountByUserID(userID, organizationID)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread notification count: %w", err)
	}
	return count, nil
}

// MarkAsRead marks a specific notification as read
func (s *notificationService) MarkAsRead(notificationID, userID uuid.UUID) error {
	// Get notification to verify ownership
	notification, err := s.notificationRepo.GetByID(notificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("notification not found")
		}
		return fmt.Errorf("failed to get notification: %w", err)
	}

	// Check if user owns this notification
	if notification.UserID != userID {
		return errors.New("notification does not belong to user")
	}

	// Mark as read
	if err := s.notificationRepo.MarkAsRead(notificationID); err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

// MarkAllAsRead marks all notifications for a user as read
func (s *notificationService) MarkAllAsRead(userID uuid.UUID) error {
	if err := s.notificationRepo.MarkAllAsRead(userID); err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}

// DeleteNotification removes a notification for a user
func (s *notificationService) DeleteNotification(notificationID, userID uuid.UUID) error {
	// Get notification to verify ownership
	notification, err := s.notificationRepo.GetByID(notificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("notification not found")
		}
		return fmt.Errorf("failed to get notification: %w", err)
	}

	// Check if user owns this notification
	if notification.UserID != userID {
		return errors.New("notification does not belong to user")
	}

	// Delete notification
	if err := s.notificationRepo.Delete(notificationID); err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	return nil
}

// DeleteAllNotifications removes all notifications for a user
func (s *notificationService) DeleteAllNotifications(userID uuid.UUID) error {
	if err := s.notificationRepo.DeleteByUserID(userID); err != nil {
		return fmt.Errorf("failed to delete all notifications: %w", err)
	}
	return nil
}
