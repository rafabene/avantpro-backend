package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// NotificationPreferenceRepository defines the interface for notification preference data operations
type NotificationPreferenceRepository interface {
	// Create inserts a new notification preference into the database
	Create(preference *models.NotificationPreference) error

	// CreateDefaults creates default notification preferences for a user
	CreateDefaults(userID uuid.UUID) error

	// GetByUserID retrieves all notification preferences for a specific user
	GetByUserID(userID uuid.UUID) ([]models.NotificationPreference, error)

	// GetByUserIDAndEvent retrieves a specific notification preference by user ID and event type
	GetByUserIDAndEvent(userID uuid.UUID, event models.NotificationEvent) (*models.NotificationPreference, error)

	// Update modifies an existing notification preference in the database
	Update(preference *models.NotificationPreference) error

	// BulkUpdate updates multiple notification preferences for a user
	BulkUpdate(userID uuid.UUID, preferences []models.NotificationPreferenceBulkItem) error

	// Delete removes a notification preference from the database (soft delete)
	Delete(id uuid.UUID) error

	// DeleteByUserID removes all notification preferences for a user (soft delete)
	DeleteByUserID(userID uuid.UUID) error

	// IsEventEnabledForUser checks if a specific event is enabled for a user
	IsEventEnabledForUser(userID uuid.UUID, event models.NotificationEvent) (bool, error)

	// GetEnabledEventsForUser retrieves all enabled events for a user
	GetEnabledEventsForUser(userID uuid.UUID) ([]models.NotificationEvent, error)
}

// notificationPreferenceRepository implements the NotificationPreferenceRepository interface
type notificationPreferenceRepository struct {
	db *gorm.DB
}

// NewNotificationPreferenceRepository creates a new NotificationPreferenceRepository instance
func NewNotificationPreferenceRepository(db *gorm.DB) NotificationPreferenceRepository {
	return &notificationPreferenceRepository{db: db}
}

// Create inserts a new notification preference into the database
func (r *notificationPreferenceRepository) Create(preference *models.NotificationPreference) error {
	return r.db.Create(preference).Error
}

// CreateDefaults creates default notification preferences for a user
func (r *notificationPreferenceRepository) CreateDefaults(userID uuid.UUID) error {
	// Check if preferences already exist for this user
	var count int64
	if err := r.db.Model(&models.NotificationPreference{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return err
	}

	// If preferences already exist, don't create defaults
	if count > 0 {
		return nil
	}

	// Create default preferences
	defaultPreferences := models.GetDefaultNotificationPreferences(userID)

	// Use a transaction to ensure all preferences are created or none
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, preference := range defaultPreferences {
			if err := tx.Create(&preference).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetByUserID retrieves all notification preferences for a specific user
func (r *notificationPreferenceRepository) GetByUserID(userID uuid.UUID) ([]models.NotificationPreference, error) {
	var preferences []models.NotificationPreference
	err := r.db.Where("user_id = ?", userID).Order("event ASC").Find(&preferences).Error
	if err != nil {
		return nil, err
	}
	return preferences, nil
}

// GetByUserIDAndEvent retrieves a specific notification preference by user ID and event type
func (r *notificationPreferenceRepository) GetByUserIDAndEvent(userID uuid.UUID, event models.NotificationEvent) (*models.NotificationPreference, error) {
	var preference models.NotificationPreference
	err := r.db.Where("user_id = ? AND event = ?", userID, event).First(&preference).Error
	if err != nil {
		return nil, err
	}
	return &preference, nil
}

// Update modifies an existing notification preference in the database
func (r *notificationPreferenceRepository) Update(preference *models.NotificationPreference) error {
	return r.db.Save(preference).Error
}

// BulkUpdate updates multiple notification preferences for a user
func (r *notificationPreferenceRepository) BulkUpdate(userID uuid.UUID, preferences []models.NotificationPreferenceBulkItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, pref := range preferences {
			// Find existing preference or create new one
			var existing models.NotificationPreference
			err := tx.Where("user_id = ? AND event = ?", userID, pref.Event).First(&existing).Error

			if err == gorm.ErrRecordNotFound {
				// Create new preference
				newPref := models.NotificationPreference{
					UserID:  userID,
					Event:   pref.Event,
					Enabled: pref.Enabled,
				}
				if err := tx.Create(&newPref).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				// Update existing preference
				existing.Enabled = pref.Enabled
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// Delete removes a notification preference from the database (soft delete)
func (r *notificationPreferenceRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.NotificationPreference{}, id).Error
}

// DeleteByUserID removes all notification preferences for a user (soft delete)
func (r *notificationPreferenceRepository) DeleteByUserID(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.NotificationPreference{}).Error
}

// IsEventEnabledForUser checks if a specific event is enabled for a user
func (r *notificationPreferenceRepository) IsEventEnabledForUser(userID uuid.UUID, event models.NotificationEvent) (bool, error) {
	var preference models.NotificationPreference
	err := r.db.Where("user_id = ? AND event = ?", userID, event).First(&preference).Error

	if err == gorm.ErrRecordNotFound {
		// If no preference found, assume enabled by default
		return true, nil
	} else if err != nil {
		return false, err
	}

	return preference.Enabled, nil
}

// GetEnabledEventsForUser retrieves all enabled events for a user
func (r *notificationPreferenceRepository) GetEnabledEventsForUser(userID uuid.UUID) ([]models.NotificationEvent, error) {
	var preferences []models.NotificationPreference
	err := r.db.Where("user_id = ? AND enabled = ?", userID, true).Find(&preferences).Error
	if err != nil {
		return nil, err
	}

	events := make([]models.NotificationEvent, len(preferences))
	for i, pref := range preferences {
		events[i] = pref.Event
	}

	return events, nil
}
