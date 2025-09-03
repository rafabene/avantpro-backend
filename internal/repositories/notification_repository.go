package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// NotificationRepository defines the interface for notification data operations
type NotificationRepository interface {
	// Create inserts a new notification into the database
	Create(notification *models.Notification) error

	// GetByID retrieves a notification by its unique identifier
	GetByID(id uuid.UUID) (*models.Notification, error)

	// GetByUserID retrieves all notifications for a specific user with pagination and sorting
	GetByUserID(userID uuid.UUID, organizationID *uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.Notification, int64, error)

	// GetUnreadByUserID retrieves all unread notifications for a specific user
	GetUnreadByUserID(userID uuid.UUID, organizationID *uuid.UUID) ([]models.Notification, error)

	// GetUnreadCountByUserID retrieves the count of unread notifications for a specific user
	GetUnreadCountByUserID(userID uuid.UUID, organizationID *uuid.UUID) (int64, error)

	// Update modifies an existing notification in the database
	Update(notification *models.Notification) error

	// MarkAsRead marks a notification as read
	MarkAsRead(id uuid.UUID) error

	// MarkAllAsRead marks all notifications for a user as read
	MarkAllAsRead(userID uuid.UUID) error

	// Delete removes a notification from the database (soft delete)
	Delete(id uuid.UUID) error

	// DeleteByUserID removes all notifications for a user (soft delete)
	DeleteByUserID(userID uuid.UUID) error
}

// notificationRepository implements the NotificationRepository interface
type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository creates a new NotificationRepository instance
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

// Create inserts a new notification into the database
func (r *notificationRepository) Create(notification *models.Notification) error {
	return r.db.Create(notification).Error
}

// GetByID retrieves a notification by its unique identifier
func (r *notificationRepository) GetByID(id uuid.UUID) (*models.Notification, error) {
	var notification models.Notification
	err := r.db.Preload("User").Preload("Organization").Where("id = ?", id).First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetByUserID retrieves all notifications for a specific user with pagination and sorting
func (r *notificationRepository) GetByUserID(userID uuid.UUID, organizationID *uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var total int64

	// Build where condition
	whereCondition := "user_id = ?"
	whereArgs := []interface{}{userID}

	if organizationID != nil {
		whereCondition += " AND organization_id = ?"
		whereArgs = append(whereArgs, *organizationID)
	}

	// Count total records for the user
	if err := r.db.Model(&models.Notification{}).Where(whereCondition, whereArgs...).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build base query
	query := r.db.Preload("Organization").Where(whereCondition, whereArgs...).Limit(limit).Offset(offset)

	// Apply sorting with validation
	allowedFields := map[string]bool{
		"title":      true,
		"type":       true,
		"read":       true,
		"created_at": true,
		"updated_at": true,
	}

	if allowedFields[sortBy] && (sortOrder == "asc" || sortOrder == "desc") {
		query = query.Order(sortBy + " " + sortOrder)
	} else {
		// Default sorting: unread first, then by creation date desc
		query = query.Order("read ASC, created_at DESC")
	}

	// Execute query
	if err := query.Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// GetUnreadByUserID retrieves all unread notifications for a specific user
func (r *notificationRepository) GetUnreadByUserID(userID uuid.UUID, organizationID *uuid.UUID) ([]models.Notification, error) {
	var notifications []models.Notification

	// Build where condition
	whereCondition := "user_id = ? AND read = ?"
	whereArgs := []interface{}{userID, false}

	if organizationID != nil {
		whereCondition += " AND organization_id = ?"
		whereArgs = append(whereArgs, *organizationID)
	}

	err := r.db.Preload("Organization").
		Where(whereCondition, whereArgs...).
		Order("created_at DESC").
		Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	return notifications, nil
}

// GetUnreadCountByUserID retrieves the count of unread notifications for a specific user
func (r *notificationRepository) GetUnreadCountByUserID(userID uuid.UUID, organizationID *uuid.UUID) (int64, error) {
	var count int64

	// Build where condition
	whereCondition := "user_id = ? AND read = ?"
	whereArgs := []interface{}{userID, false}

	if organizationID != nil {
		whereCondition += " AND organization_id = ?"
		whereArgs = append(whereArgs, *organizationID)
	}

	err := r.db.Model(&models.Notification{}).
		Where(whereCondition, whereArgs...).
		Count(&count).Error
	return count, err
}

// Update modifies an existing notification in the database
func (r *notificationRepository) Update(notification *models.Notification) error {
	return r.db.Save(notification).Error
}

// MarkAsRead marks a notification as read
func (r *notificationRepository) MarkAsRead(id uuid.UUID) error {
	return r.db.Model(&models.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": gorm.Expr("NOW()"),
		}).Error
}

// MarkAllAsRead marks all notifications for a user as read
func (r *notificationRepository) MarkAllAsRead(userID uuid.UUID) error {
	return r.db.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": gorm.Expr("NOW()"),
		}).Error
}

// Delete removes a notification from the database (soft delete)
func (r *notificationRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Notification{}, id).Error
}

// DeleteByUserID removes all notifications for a user (soft delete)
func (r *notificationRepository) DeleteByUserID(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.Notification{}).Error
}
