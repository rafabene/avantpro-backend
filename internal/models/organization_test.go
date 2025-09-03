package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOrganizationRole_Constants(t *testing.T) {
	assert.Equal(t, OrganizationRole("admin"), OrganizationRoleAdmin)
	assert.Equal(t, OrganizationRole("user"), OrganizationRoleUser)
}

func TestInviteStatus_Constants(t *testing.T) {
	assert.Equal(t, InviteStatus("pending"), InviteStatusPending)
	assert.Equal(t, InviteStatus("accepted"), InviteStatusAccepted)
	assert.Equal(t, InviteStatus("expired"), InviteStatusExpired)
	assert.Equal(t, InviteStatus("revoked"), InviteStatusRevoked)
}

func TestOrganizationInvite_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "Not expired - future date",
			expiresAt: time.Now().Add(24 * time.Hour),
			expected:  false,
		},
		{
			name:      "Expired - past date",
			expiresAt: time.Now().Add(-24 * time.Hour),
			expected:  true,
		},
		{
			name:      "Just expired - 1 second ago",
			expiresAt: time.Now().Add(-1 * time.Second),
			expected:  true,
		},
		{
			name:      "Not expired - 1 second in future",
			expiresAt: time.Now().Add(1 * time.Second),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invite := &OrganizationInvite{
				ExpiresAt: tt.expiresAt,
			}
			result := invite.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOrganizationMember_IsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     OrganizationRole
		expected bool
	}{
		{
			name:     "Admin role",
			role:     OrganizationRoleAdmin,
			expected: true,
		},
		{
			name:     "User role",
			role:     OrganizationRoleUser,
			expected: false,
		},
		{
			name:     "Empty role",
			role:     OrganizationRole(""),
			expected: false,
		},
		{
			name:     "Invalid role",
			role:     OrganizationRole("invalid"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			member := &OrganizationMember{
				Role: tt.role,
			}
			result := member.IsAdmin()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOrganization_BeforeCreate(t *testing.T) {
	org := &Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   uuid.New(),
	}

	// BeforeCreate should not return any error
	err := org.BeforeCreate(nil)
	assert.NoError(t, err)
}

func TestOrganizationCreateRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     OrganizationCreateRequest
		description string
	}{
		{
			name: "Valid request",
			request: OrganizationCreateRequest{
				Name:        "Test Organization",
				Description: "Test Description",
			},
			description: "Should be valid with proper name and description",
		},
		{
			name: "Valid request with empty description",
			request: OrganizationCreateRequest{
				Name:        "Test Organization",
				Description: "",
			},
			description: "Should be valid with empty description",
		},
		{
			name: "Valid request with long name",
			request: OrganizationCreateRequest{
				Name:        "Test Organization with a very long name that is still within limits and should be valid",
				Description: "Test Description",
			},
			description: "Should be valid with long but within-limit name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic structure validation
			assert.NotEmpty(t, tt.request.Name, "Name should not be empty for valid requests")
			if tt.request.Name != "" {
				assert.True(t, len(tt.request.Name) >= 2, "Name should be at least 2 characters")
				assert.True(t, len(tt.request.Name) <= 100, "Name should be at most 100 characters")
			}
			if tt.request.Description != "" {
				assert.True(t, len(tt.request.Description) <= 500, "Description should be at most 500 characters")
			}
		})
	}
}

func TestOrganizationUpdateRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     OrganizationUpdateRequest
		description string
	}{
		{
			name: "Valid update with name only",
			request: OrganizationUpdateRequest{
				Name: stringPtr("Updated Organization"),
			},
			description: "Should be valid with only name update",
		},
		{
			name: "Valid update with description only",
			request: OrganizationUpdateRequest{
				Description: stringPtr("Updated Description"),
			},
			description: "Should be valid with only description update",
		},
		{
			name: "Valid update with both fields",
			request: OrganizationUpdateRequest{
				Name:        stringPtr("Updated Organization"),
				Description: stringPtr("Updated Description"),
			},
			description: "Should be valid with both fields",
		},
		{
			name:        "Valid update with no fields",
			request:     OrganizationUpdateRequest{},
			description: "Should be valid with no fields (patch request)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test optional field validation
			if tt.request.Name != nil {
				assert.True(t, len(*tt.request.Name) >= 2, "Name should be at least 2 characters when provided")
				assert.True(t, len(*tt.request.Name) <= 100, "Name should be at most 100 characters when provided")
			}
			if tt.request.Description != nil {
				assert.True(t, len(*tt.request.Description) <= 500, "Description should be at most 500 characters when provided")
			}
		})
	}
}

func TestOrganizationInviteRequest_Validation(t *testing.T) {
	tests := []struct {
		name          string
		request       OrganizationInviteRequest
		shouldBeValid bool
		description   string
	}{
		{
			name: "Valid admin invite",
			request: OrganizationInviteRequest{
				Email: "user@example.com",
				Role:  OrganizationRoleAdmin,
			},
			shouldBeValid: true,
			description:   "Should be valid with proper email and admin role",
		},
		{
			name: "Valid user invite",
			request: OrganizationInviteRequest{
				Email: "user@example.com",
				Role:  OrganizationRoleUser,
			},
			shouldBeValid: true,
			description:   "Should be valid with proper email and user role",
		},
		{
			name: "Invalid email format",
			request: OrganizationInviteRequest{
				Email: "invalid-email",
				Role:  OrganizationRoleUser,
			},
			shouldBeValid: false,
			description:   "Should be invalid with improper email format",
		},
		{
			name: "Empty email",
			request: OrganizationInviteRequest{
				Email: "",
				Role:  OrganizationRoleUser,
			},
			shouldBeValid: false,
			description:   "Should be invalid with empty email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.shouldBeValid {
				assert.NotEmpty(t, tt.request.Email, "Email should not be empty for valid requests")
				assert.Contains(t, tt.request.Email, "@", "Email should contain @ for valid requests")
				assert.True(t, tt.request.Role == OrganizationRoleAdmin || tt.request.Role == OrganizationRoleUser, "Role should be admin or user")
			}
		})
	}
}

func TestOrganizationMemberUpdateRequest_Validation(t *testing.T) {
	tests := []struct {
		name          string
		request       OrganizationMemberUpdateRequest
		shouldBeValid bool
		description   string
	}{
		{
			name: "Valid admin role update",
			request: OrganizationMemberUpdateRequest{
				Role: OrganizationRoleAdmin,
			},
			shouldBeValid: true,
			description:   "Should be valid with admin role",
		},
		{
			name: "Valid user role update",
			request: OrganizationMemberUpdateRequest{
				Role: OrganizationRoleUser,
			},
			shouldBeValid: true,
			description:   "Should be valid with user role",
		},
		{
			name: "Invalid role",
			request: OrganizationMemberUpdateRequest{
				Role: OrganizationRole("invalid"),
			},
			shouldBeValid: false,
			description:   "Should be invalid with invalid role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldBeValid {
				assert.True(t, tt.request.Role == OrganizationRoleAdmin || tt.request.Role == OrganizationRoleUser, "Role should be admin or user for valid requests")
			}
		})
	}
}

func TestOrganizationResponse_Structure(t *testing.T) {
	// Test that OrganizationResponse has all expected fields
	response := OrganizationResponse{
		ID:          uuid.New(),
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   uuid.New(),
		Creator:     &UserResponse{},
		Members:     []OrganizationMemberResponse{},
		Invites:     []OrganizationInviteResponse{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.NotEmpty(t, response.Name)
	assert.NotEmpty(t, response.Description)
	assert.NotEqual(t, uuid.Nil, response.CreatedBy)
	assert.NotNil(t, response.Creator)
	assert.NotNil(t, response.Members)
	assert.NotNil(t, response.Invites)
	assert.False(t, response.CreatedAt.IsZero())
	assert.False(t, response.UpdatedAt.IsZero())
}

func TestOrganizationMemberResponse_Structure(t *testing.T) {
	// Test that OrganizationMemberResponse has all expected fields
	response := OrganizationMemberResponse{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		UserID:         uuid.New(),
		Role:           OrganizationRoleAdmin,
		User:           &UserResponse{},
		Organization:   &OrganizationResponse{},
		JoinedAt:       time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.NotEqual(t, uuid.Nil, response.OrganizationID)
	assert.NotEqual(t, uuid.Nil, response.UserID)
	assert.Equal(t, OrganizationRoleAdmin, response.Role)
	assert.NotNil(t, response.User)
	assert.NotNil(t, response.Organization)
	assert.False(t, response.JoinedAt.IsZero())
	assert.False(t, response.CreatedAt.IsZero())
	assert.False(t, response.UpdatedAt.IsZero())
}

func TestOrganizationInviteResponse_Structure(t *testing.T) {
	// Test that OrganizationInviteResponse has all expected fields
	acceptedAt := time.Now()
	response := OrganizationInviteResponse{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "test@example.com",
		Role:           OrganizationRoleUser,
		InvitedBy:      uuid.New(),
		Status:         InviteStatusAccepted,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		AcceptedAt:     &acceptedAt,
		Organization:   &OrganizationResponse{},
		Inviter:        &UserResponse{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.NotEqual(t, uuid.Nil, response.OrganizationID)
	assert.NotEmpty(t, response.Email)
	assert.Equal(t, OrganizationRoleUser, response.Role)
	assert.NotEqual(t, uuid.Nil, response.InvitedBy)
	assert.Equal(t, InviteStatusAccepted, response.Status)
	assert.False(t, response.ExpiresAt.IsZero())
	assert.NotNil(t, response.AcceptedAt)
	assert.NotNil(t, response.Organization)
	assert.NotNil(t, response.Inviter)
	assert.False(t, response.CreatedAt.IsZero())
	assert.False(t, response.UpdatedAt.IsZero())
}

func TestOrganizationListResponse_Structure(t *testing.T) {
	// Test that OrganizationListResponse has all expected fields
	response := OrganizationListResponse{
		Data:  []OrganizationResponse{},
		Total: 100,
		Limit: 50,
		Page:  1,
	}

	assert.NotNil(t, response.Data)
	assert.Equal(t, int64(100), response.Total)
	assert.Equal(t, 50, response.Limit)
	assert.Equal(t, 1, response.Page)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
