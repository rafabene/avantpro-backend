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
// This interface provides methods for managing organization notification preferences.
type NotificationPreferenceService interface {
	// GetOrganizationPreferences retrieves all notification preferences for an organization.
	// If no preferences exist, creates and returns default preferences.
	// Parameters:
	//   - organizationID: ID of the organization to get preferences for
	// Returns:
	//   - []NotificationPreferenceResponse: List of organization preferences
	//   - error: Error if retrieval or creation fails
	GetOrganizationPreferences(organizationID uuid.UUID) ([]NotificationPreferenceResponse, error)

	// UpdateOrganizationPreferences updates notification preferences for an organization using bulk update.
	// This method allows updating multiple preferences at once.
	// Parameters:
	//   - organizationID: ID of the organization to update preferences for
	//   - req: Bulk update request containing preference updates
	// Returns:
	//   - []NotificationPreferenceResponse: Updated preferences
	//   - error: Error if validation fails or update fails
	UpdateOrganizationPreferences(organizationID uuid.UUID, req *NotificationPreferenceBulkUpdateRequest) ([]NotificationPreferenceResponse, error)

	// UpdateSinglePreference updates a single notification preference.
	// This method allows updating individual preference settings.
	// Parameters:
	//   - organizationID: ID of the organization
	//   - event: The notification event type to update
	//   - req: Update request containing new preference values
	// Returns:
	//   - *NotificationPreferenceResponse: Updated preference
	//   - error: Error if preference not found or update fails
	UpdateSinglePreference(organizationID uuid.UUID, event NotificationEvent, req *NotificationPreferenceUpdateRequest) (*NotificationPreferenceResponse, error)

	// ResetToDefaults resets all organization preferences to default values.
	// This method deletes existing preferences and creates new default ones.
	// Parameters:
	//   - organizationID: ID of the organization to reset preferences for
	// Returns:
	//   - []NotificationPreferenceResponse: Default preferences
	//   - error: Error if reset fails
	ResetToDefaults(organizationID uuid.UUID) ([]NotificationPreferenceResponse, error)

	// IsEventEnabledForOrganization checks if a specific notification event is enabled for an organization.
	// This method is used by the notification service to check if notifications should be sent.
	// Parameters:
	//   - organizationID: ID of the organization to check
	//   - event: The notification event type to check
	// Returns:
	//   - bool: True if the event is enabled for the organization
	//   - error: Error if check fails
	IsEventEnabledForOrganization(organizationID uuid.UUID, event NotificationEvent) (bool, error)

	// GetAvailableEvents returns all available notification events with descriptions.
	// This method provides metadata about notification types for UI display.
	// Returns:
	//   - map[NotificationEvent]string: Event to description mapping
	GetAvailableEvents() map[NotificationEvent]string

	// GenerateTestNotification creates a test notification for an organization.
	// This method allows users to test their notification settings.
	// Parameters:
	//   - userID: ID of the user sending the test
	//   - req: Test notification request containing title, message, and type
	// Returns:
	//   - *NotificationResponse: Created test notification
	//   - error: Error if creation fails
	GenerateTestNotification(userID uuid.UUID, req *TestNotificationRequest) (*NotificationResponse, error)
}

// notificationPreferenceService implements the NotificationPreferenceService interface
type notificationPreferenceService struct {
	preferenceRepo      repositories.NotificationPreferenceRepository
	notificationRepo    repositories.NotificationRepository
	organizationRepo    repositories.OrganizationRepositoryInterface
	notificationService NotificationService
}

// NewNotificationPreferenceService creates a new NotificationPreferenceService instance
func NewNotificationPreferenceService(
	preferenceRepo repositories.NotificationPreferenceRepository,
	notificationRepo repositories.NotificationRepository,
	organizationRepo repositories.OrganizationRepositoryInterface,
	notificationService NotificationService,
) NotificationPreferenceService {
	return &notificationPreferenceService{
		preferenceRepo:      preferenceRepo,
		notificationRepo:    notificationRepo,
		organizationRepo:    organizationRepo,
		notificationService: notificationService,
	}
}

// GetOrganizationPreferences retrieves all notification preferences for an organization
func (s *notificationPreferenceService) GetOrganizationPreferences(organizationID uuid.UUID) ([]NotificationPreferenceResponse, error) {
	// Validate organization exists
	_, err := s.organizationRepo.GetByID(organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to validate organization: %w", err)
	}

	// Get existing preferences
	preferences, err := s.preferenceRepo.GetByOrganizationID(organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization preferences: %w", err)
	}

	// If no preferences exist, create defaults
	if len(preferences) == 0 {
		if err := s.preferenceRepo.CreateDefaults(organizationID); err != nil {
			return nil, fmt.Errorf("failed to create default preferences: %w", err)
		}

		// Retrieve the newly created defaults
		preferences, err = s.preferenceRepo.GetByOrganizationID(organizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get default preferences: %w", err)
		}
	}

	// Convert to response format
	responses := make([]NotificationPreferenceResponse, len(preferences))
	for i, pref := range preferences {
		responses[i] = NotificationPreferenceResponse{
			ID:        pref.ID,
			Event:     NotificationEvent(pref.Event),
			Enabled:   pref.Enabled,
			CreatedAt: pref.CreatedAt,
			UpdatedAt: pref.UpdatedAt,
		}
	}

	return responses, nil
}

// UpdateOrganizationPreferences updates notification preferences for an organization using bulk update
func (s *notificationPreferenceService) UpdateOrganizationPreferences(organizationID uuid.UUID, req *NotificationPreferenceBulkUpdateRequest) ([]NotificationPreferenceResponse, error) {
	// Validate organization exists
	_, err := s.organizationRepo.GetByID(organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to validate organization: %w", err)
	}

	// Validate all events are valid
	for _, pref := range req.Preferences {
		if !pref.Event.IsValid() {
			return nil, fmt.Errorf("invalid notification event: %s", pref.Event)
		}
	}

	// Convert to models format for repository
	repoPrefs := make([]models.NotificationPreferenceBulkItem, len(req.Preferences))
	for i, pref := range req.Preferences {
		repoPrefs[i] = models.NotificationPreferenceBulkItem{
			Event:   models.NotificationEvent(pref.Event),
			Enabled: pref.Enabled,
		}
	}

	// Perform bulk update
	if err := s.preferenceRepo.BulkUpdate(organizationID, repoPrefs); err != nil {
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}

	// Return updated preferences
	return s.GetOrganizationPreferences(organizationID)
}

// UpdateSinglePreference updates a single notification preference
func (s *notificationPreferenceService) UpdateSinglePreference(organizationID uuid.UUID, event NotificationEvent, req *NotificationPreferenceUpdateRequest) (*NotificationPreferenceResponse, error) {
	// Validate event
	if !event.IsValid() {
		return nil, fmt.Errorf("invalid notification event: %s", event)
	}

	// Get existing preference
	preference, err := s.preferenceRepo.GetByOrganizationIDAndEvent(organizationID, models.NotificationEvent(event))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new preference if it doesn't exist
			preference = &models.NotificationPreference{
				OrganizationID: organizationID,
				Event:          models.NotificationEvent(event),
				Enabled:        true, // Default enabled
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
	response := &NotificationPreferenceResponse{
		ID:        preference.ID,
		Event:     NotificationEvent(preference.Event),
		Enabled:   preference.Enabled,
		CreatedAt: preference.CreatedAt,
		UpdatedAt: preference.UpdatedAt,
	}

	return response, nil
}

// ResetToDefaults resets all organization preferences to default values
func (s *notificationPreferenceService) ResetToDefaults(organizationID uuid.UUID) ([]NotificationPreferenceResponse, error) {
	// Validate organization exists
	_, err := s.organizationRepo.GetByID(organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to validate organization: %w", err)
	}

	// Delete existing preferences
	if err := s.preferenceRepo.DeleteByOrganizationID(organizationID); err != nil {
		return nil, fmt.Errorf("failed to delete existing preferences: %w", err)
	}

	// Create default preferences
	if err := s.preferenceRepo.CreateDefaults(organizationID); err != nil {
		return nil, fmt.Errorf("failed to create default preferences: %w", err)
	}

	// Return new preferences
	return s.GetOrganizationPreferences(organizationID)
}

// IsEventEnabledForOrganization checks if a specific notification event is enabled for an organization
func (s *notificationPreferenceService) IsEventEnabledForOrganization(organizationID uuid.UUID, event NotificationEvent) (bool, error) {
	return s.preferenceRepo.IsEventEnabledForOrganization(organizationID, models.NotificationEvent(event))
}

// GetAvailableEvents returns all available notification events with descriptions
func (s *notificationPreferenceService) GetAvailableEvents() map[NotificationEvent]string {
	events := []NotificationEvent{
		NotificationEventMemberJoined,
		NotificationEventMemberLeft,
		NotificationEventMemberRoleChanged,
		NotificationEventInvitationSent,
		NotificationEventInvitationAccepted,
		NotificationEventInvitationExpired,
		NotificationEventOrganizationUpdate,
	}

	result := make(map[NotificationEvent]string)
	for _, event := range events {
		result[event] = event.GetDescription()
	}

	return result
}

// GenerateTestNotification creates a test notification for a user
func (s *notificationPreferenceService) GenerateTestNotification(userID uuid.UUID, req *TestNotificationRequest) (*NotificationResponse, error) {
	// Create notification request
	createReq := &CreateNotificationRequest{
		UserID:         userID,
		OrganizationID: req.OrganizationID,
		Title:          req.Title,
		Message:        req.Message,
		Type:           req.Type,
		Data:           "{\"type\":\"test\",\"generated_by\":\"user\"}",
	}

	// Use NotificationService to create notification
	return s.notificationService.CreateNotification(createReq)
}
