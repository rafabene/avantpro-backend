package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/repositories"
)

// NotificationPreferenceService interface defines notification preference management operations.
// This interface provides methods for managing user notification preferences.
type NotificationPreferenceService interface {
	// GetUserPreferences retrieves all notification preferences for a user.
	// If no preferences exist, creates and returns default preferences.
	// Parameters:
	//   - userID: ID of the user to get preferences for
	// Returns:
	//   - []models.NotificationPreferenceResponse: List of user preferences
	//   - error: Error if retrieval or creation fails
	GetUserPreferences(userID uuid.UUID) ([]models.NotificationPreferenceResponse, error)

	// UpdateUserPreferences updates notification preferences for a user using bulk update.
	// This method allows updating multiple preferences at once.
	// Parameters:
	//   - userID: ID of the user to update preferences for
	//   - req: Bulk update request containing preference updates
	// Returns:
	//   - []models.NotificationPreferenceResponse: Updated preferences
	//   - error: Error if validation fails or update fails
	UpdateUserPreferences(userID uuid.UUID, req *models.NotificationPreferenceBulkUpdateRequest) ([]models.NotificationPreferenceResponse, error)

	// UpdateSinglePreference updates a single notification preference.
	// This method allows updating individual preference settings.
	// Parameters:
	//   - userID: ID of the user (for authorization)
	//   - event: The notification event type to update
	//   - req: Update request containing new preference values
	// Returns:
	//   - *models.NotificationPreferenceResponse: Updated preference
	//   - error: Error if preference not found or update fails
	UpdateSinglePreference(userID uuid.UUID, event models.NotificationEvent, req *models.NotificationPreferenceUpdateRequest) (*models.NotificationPreferenceResponse, error)

	// ResetToDefaults resets all user preferences to default values.
	// This method deletes existing preferences and creates new default ones.
	// Parameters:
	//   - userID: ID of the user to reset preferences for
	// Returns:
	//   - []models.NotificationPreferenceResponse: Default preferences
	//   - error: Error if reset fails
	ResetToDefaults(userID uuid.UUID) ([]models.NotificationPreferenceResponse, error)

	// IsEventEnabledForUser checks if a specific notification event is enabled for a user.
	// This method is used by the notification service to check if notifications should be sent.
	// Parameters:
	//   - userID: ID of the user to check
	//   - event: The notification event type to check
	// Returns:
	//   - bool: True if the event is enabled for the user
	//   - error: Error if check fails
	IsEventEnabledForUser(userID uuid.UUID, event models.NotificationEvent) (bool, error)

	// GetAvailableEvents returns all available notification events with descriptions.
	// This method provides metadata about notification types for UI display.
	// Returns:
	//   - map[models.NotificationEvent]string: Event to description mapping
	GetAvailableEvents() map[models.NotificationEvent]string

	// GenerateTestNotification creates a test notification for a user.
	// This method allows users to test their notification settings.
	// Parameters:
	//   - userID: ID of the user to send test notification to
	//   - req: Test notification request containing title, message, and type
	// Returns:
	//   - *models.NotificationResponse: Created test notification
	//   - error: Error if creation fails
	GenerateTestNotification(userID uuid.UUID, req *models.TestNotificationRequest) (*models.NotificationResponse, error)
}

// notificationPreferenceService implements the NotificationPreferenceService interface
type notificationPreferenceService struct {
	preferenceRepo      repositories.NotificationPreferenceRepository
	notificationRepo    repositories.NotificationRepository
	userRepo            repositories.UserRepository
	notificationService NotificationService
}

// NewNotificationPreferenceService creates a new NotificationPreferenceService instance
func NewNotificationPreferenceService(
	preferenceRepo repositories.NotificationPreferenceRepository,
	notificationRepo repositories.NotificationRepository,
	userRepo repositories.UserRepository,
	notificationService NotificationService,
) NotificationPreferenceService {
	return &notificationPreferenceService{
		preferenceRepo:      preferenceRepo,
		notificationRepo:    notificationRepo,
		userRepo:            userRepo,
		notificationService: notificationService,
	}
}

// GetUserPreferences retrieves all notification preferences for a user
func (s *notificationPreferenceService) GetUserPreferences(userID uuid.UUID) ([]models.NotificationPreferenceResponse, error) {
	// Validate user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to validate user: %w", err)
	}

	// Get existing preferences
	preferences, err := s.preferenceRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	// If no preferences exist, create defaults
	if len(preferences) == 0 {
		if err := s.preferenceRepo.CreateDefaults(userID); err != nil {
			return nil, fmt.Errorf("failed to create default preferences: %w", err)
		}

		// Retrieve the newly created defaults
		preferences, err = s.preferenceRepo.GetByUserID(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get default preferences: %w", err)
		}
	}

	// Convert to response format
	responses := make([]models.NotificationPreferenceResponse, len(preferences))
	for i, pref := range preferences {
		responses[i] = models.NotificationPreferenceResponse{
			ID:        pref.ID,
			Event:     pref.Event,
			Enabled:   pref.Enabled,
			CreatedAt: pref.CreatedAt,
			UpdatedAt: pref.UpdatedAt,
		}
	}

	return responses, nil
}

// UpdateUserPreferences updates notification preferences for a user using bulk update
func (s *notificationPreferenceService) UpdateUserPreferences(userID uuid.UUID, req *models.NotificationPreferenceBulkUpdateRequest) ([]models.NotificationPreferenceResponse, error) {
	// Validate user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to validate user: %w", err)
	}

	// Validate all events are valid
	for _, pref := range req.Preferences {
		if !pref.Event.IsValid() {
			return nil, fmt.Errorf("invalid notification event: %s", pref.Event)
		}
	}

	// Perform bulk update
	if err := s.preferenceRepo.BulkUpdate(userID, req.Preferences); err != nil {
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}

	// Return updated preferences
	return s.GetUserPreferences(userID)
}

// UpdateSinglePreference updates a single notification preference
func (s *notificationPreferenceService) UpdateSinglePreference(userID uuid.UUID, event models.NotificationEvent, req *models.NotificationPreferenceUpdateRequest) (*models.NotificationPreferenceResponse, error) {
	// Validate event
	if !event.IsValid() {
		return nil, fmt.Errorf("invalid notification event: %s", event)
	}

	// Get existing preference
	preference, err := s.preferenceRepo.GetByUserIDAndEvent(userID, event)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new preference if it doesn't exist
			preference = &models.NotificationPreference{
				UserID:  userID,
				Event:   event,
				Enabled: true, // Default enabled
			}
		} else {
			return nil, fmt.Errorf("failed to get preference: %w", err)
		}
	}

	// Update fields if provided
	if req.Enabled != nil {
		preference.Enabled = *req.Enabled
	}

	// Save preference
	if err := s.preferenceRepo.Update(preference); err != nil {
		return nil, fmt.Errorf("failed to update preference: %w", err)
	}

	// Convert to response
	response := &models.NotificationPreferenceResponse{
		ID:        preference.ID,
		Event:     preference.Event,
		Enabled:   preference.Enabled,
		CreatedAt: preference.CreatedAt,
		UpdatedAt: preference.UpdatedAt,
	}

	return response, nil
}

// ResetToDefaults resets all user preferences to default values
func (s *notificationPreferenceService) ResetToDefaults(userID uuid.UUID) ([]models.NotificationPreferenceResponse, error) {
	// Validate user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to validate user: %w", err)
	}

	// Delete existing preferences
	if err := s.preferenceRepo.DeleteByUserID(userID); err != nil {
		return nil, fmt.Errorf("failed to delete existing preferences: %w", err)
	}

	// Create default preferences
	if err := s.preferenceRepo.CreateDefaults(userID); err != nil {
		return nil, fmt.Errorf("failed to create default preferences: %w", err)
	}

	// Return new preferences
	return s.GetUserPreferences(userID)
}

// IsEventEnabledForUser checks if a specific notification event is enabled for a user
func (s *notificationPreferenceService) IsEventEnabledForUser(userID uuid.UUID, event models.NotificationEvent) (bool, error) {
	return s.preferenceRepo.IsEventEnabledForUser(userID, event)
}

// GetAvailableEvents returns all available notification events with descriptions
func (s *notificationPreferenceService) GetAvailableEvents() map[models.NotificationEvent]string {
	events := []models.NotificationEvent{
		models.NotificationEventMemberJoined,
		models.NotificationEventMemberLeft,
		models.NotificationEventMemberRoleChanged,
		models.NotificationEventInvitationSent,
		models.NotificationEventInvitationAccepted,
		models.NotificationEventInvitationExpired,
		models.NotificationEventOrganizationUpdate,
	}

	result := make(map[models.NotificationEvent]string)
	for _, event := range events {
		result[event] = event.GetDescription()
	}

	return result
}

// GenerateTestNotification creates a test notification for a user
func (s *notificationPreferenceService) GenerateTestNotification(userID uuid.UUID, req *models.TestNotificationRequest) (*models.NotificationResponse, error) {
	// Create notification request
	createReq := &models.CreateNotificationRequest{
		UserID:         userID,
		OrganizationID: req.OrganizationID,
		Title:          req.Title,
		Message:        req.Message,
		Type:           req.Type,
		Data:           "{\"type\":\"test\",\"generated_by\":\"user\"}",
	}

	// Use NotificationService to create notification (which will handle WebSocket broadcast)
	return s.notificationService.CreateNotification(createReq)
}
