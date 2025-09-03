package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// MockOrganizationRepository is a mock implementation of OrganizationRepositoryInterface
type MockOrganizationRepository struct {
	mock.Mock
}

func (m *MockOrganizationRepository) Create(org *models.Organization) error {
	args := m.Called(org)
	return args.Error(0)
}

func (m *MockOrganizationRepository) GetByID(id uuid.UUID) (*models.Organization, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetByCreator(creatorID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.Organization, int64, error) {
	args := m.Called(creatorID, limit, offset, sortBy, sortOrder)
	return args.Get(0).([]models.Organization), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrganizationRepository) List(limit, offset int, sortBy, sortOrder string) ([]models.Organization, int64, error) {
	args := m.Called(limit, offset, sortBy, sortOrder)
	return args.Get(0).([]models.Organization), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrganizationRepository) Update(org *models.Organization) error {
	args := m.Called(org)
	return args.Error(0)
}

func (m *MockOrganizationRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockOrganizationRepository) GetMember(orgID, userID uuid.UUID) (*models.OrganizationMember, error) {
	args := m.Called(orgID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationRepository) GetMembers(orgID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error) {
	args := m.Called(orgID, limit, offset, sortBy, sortOrder)
	return args.Get(0).([]models.OrganizationMember), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrganizationRepository) AddMember(member *models.OrganizationMember) error {
	args := m.Called(member)
	return args.Error(0)
}

func (m *MockOrganizationRepository) UpdateMember(member *models.OrganizationMember) error {
	args := m.Called(member)
	return args.Error(0)
}

func (m *MockOrganizationRepository) RemoveMember(orgID, userID uuid.UUID) error {
	args := m.Called(orgID, userID)
	return args.Error(0)
}

func (m *MockOrganizationRepository) GetUserMemberships(userID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationMember, int64, error) {
	args := m.Called(userID, limit, offset, sortBy, sortOrder)
	return args.Get(0).([]models.OrganizationMember), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrganizationRepository) CreateInvite(invite *models.OrganizationInvite) error {
	args := m.Called(invite)
	return args.Error(0)
}

func (m *MockOrganizationRepository) GetInviteByID(id uuid.UUID) (*models.OrganizationInvite, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationInvite), args.Error(1)
}

func (m *MockOrganizationRepository) GetInviteByToken(token string) (*models.OrganizationInvite, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationInvite), args.Error(1)
}

func (m *MockOrganizationRepository) GetInvites(orgID uuid.UUID, limit, offset int, sortBy, sortOrder string) ([]models.OrganizationInvite, int64, error) {
	args := m.Called(orgID, limit, offset, sortBy, sortOrder)
	return args.Get(0).([]models.OrganizationInvite), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrganizationRepository) GetPendingInviteByEmail(orgID uuid.UUID, email string) (*models.OrganizationInvite, error) {
	args := m.Called(orgID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationInvite), args.Error(1)
}

func (m *MockOrganizationRepository) UpdateInvite(invite *models.OrganizationInvite) error {
	args := m.Called(invite)
	return args.Error(0)
}

func (m *MockOrganizationRepository) RegenerateInviteToken(invite *models.OrganizationInvite) error {
	args := m.Called(invite)
	return args.Error(0)
}

func (m *MockOrganizationRepository) DeleteInvite(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockOrganizationRepository) ExpireInvites() error {
	args := m.Called()
	return args.Error(0)
}

// MockEmailService is a mock implementation of EmailServiceInterface
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendOrganizationInvite(invite *models.OrganizationInvite, baseURL string) error {
	args := m.Called(invite, baseURL)
	return args.Error(0)
}

func TestNewOrganizationService(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}

	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	assert.NotNil(t, service)
	assert.IsType(t, &OrganizationService{}, service)
}

func TestOrganizationService_CreateOrganization_Success(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	creatorID := uuid.New()
	orgID := uuid.New()

	creator := &models.User{
		ID:       creatorID,
		Username: "creator@example.com",
		Name:     "Creator User",
	}

	req := &models.OrganizationCreateRequest{
		Name:        "Test Organization",
		Description: "Test Description",
	}

	createdOrg := &models.Organization{
		ID:          orgID,
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creatorID,
		Creator:     *creator,
	}

	mockUserRepo.On("GetByID", creatorID).Return(creator, nil)
	mockOrgRepo.On("Create", mock.AnythingOfType("*models.Organization")).Return(nil).Run(func(args mock.Arguments) {
		org := args.Get(0).(*models.Organization)
		org.ID = orgID
	})
	mockOrgRepo.On("GetByID", orgID).Return(createdOrg, nil)

	result, err := service.CreateOrganization(req, creatorID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Organization", result.Name)
	assert.Equal(t, "Test Description", result.Description)
	assert.Equal(t, creatorID, result.CreatedBy)

	mockUserRepo.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_CreateOrganization_CreatorNotFound(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	creatorID := uuid.New()

	req := &models.OrganizationCreateRequest{
		Name:        "Test Organization",
		Description: "Test Description",
	}

	mockUserRepo.On("GetByID", creatorID).Return(nil, gorm.ErrRecordNotFound)

	result, err := service.CreateOrganization(req, creatorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get creator")

	mockUserRepo.AssertExpectations(t)
}

func TestOrganizationService_GetOrganization_Success(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	orgID := uuid.New()
	org := &models.Organization{
		ID:          orgID,
		Name:        "Test Organization",
		Description: "Test Description",
	}

	mockOrgRepo.On("GetByID", orgID).Return(org, nil)

	result, err := service.GetOrganization(orgID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, orgID, result.ID)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_UpdateOrganization_Success(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	orgID := uuid.New()
	creatorID := uuid.New()
	newName := "Updated Organization"
	newDescription := "Updated Description"

	org := &models.Organization{
		ID:          orgID,
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creatorID,
	}

	updatedOrg := &models.Organization{
		ID:          orgID,
		Name:        newName,
		Description: newDescription,
		CreatedBy:   creatorID,
	}

	req := &models.OrganizationUpdateRequest{
		Name:        &newName,
		Description: &newDescription,
	}

	mockOrgRepo.On("GetByID", orgID).Return(org, nil).Once()
	mockOrgRepo.On("Update", mock.AnythingOfType("*models.Organization")).Return(nil)
	mockOrgRepo.On("GetByID", orgID).Return(updatedOrg, nil).Once()

	result, err := service.UpdateOrganization(orgID, req, creatorID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, result.Name)
	assert.Equal(t, newDescription, result.Description)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_UpdateOrganization_InsufficientPermissions(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	orgID := uuid.New()
	creatorID := uuid.New()
	nonAdminID := uuid.New()

	org := &models.Organization{
		ID:        orgID,
		Name:      "Test Organization",
		CreatedBy: creatorID,
	}

	req := &models.OrganizationUpdateRequest{
		Name: func() *string { s := "Updated Name"; return &s }(),
	}

	mockOrgRepo.On("GetByID", orgID).Return(org, nil)
	mockOrgRepo.On("GetMember", orgID, nonAdminID).Return(nil, gorm.ErrRecordNotFound)

	result, err := service.UpdateOrganization(orgID, req, nonAdminID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "insufficient permissions")

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_DeleteOrganization_Success(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	orgID := uuid.New()
	creatorID := uuid.New()

	org := &models.Organization{
		ID:        orgID,
		Name:      "Test Organization",
		CreatedBy: creatorID,
	}

	mockOrgRepo.On("GetByID", orgID).Return(org, nil)
	mockOrgRepo.On("Delete", orgID).Return(nil)

	err := service.DeleteOrganization(orgID, creatorID)

	assert.NoError(t, err)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_DeleteOrganization_NotCreator(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	orgID := uuid.New()
	creatorID := uuid.New()
	otherUserID := uuid.New()

	org := &models.Organization{
		ID:        orgID,
		Name:      "Test Organization",
		CreatedBy: creatorID,
	}

	mockOrgRepo.On("GetByID", orgID).Return(org, nil)

	err := service.DeleteOrganization(orgID, otherUserID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only the creator can delete")

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_InviteUser_Success(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	orgID := uuid.New()
	inviterID := uuid.New()
	inviteID := uuid.New()

	org := &models.Organization{
		ID:        orgID,
		Name:      "Test Organization",
		CreatedBy: inviterID,
	}

	req := &models.OrganizationInviteRequest{
		Email: "invitee@example.com",
		Role:  models.OrganizationRoleUser,
	}

	invite := &models.OrganizationInvite{
		ID:             inviteID,
		OrganizationID: orgID,
		Email:          "invitee@example.com",
		Role:           models.OrganizationRoleUser,
		InvitedBy:      inviterID,
		Status:         models.InviteStatusPending,
		Organization:   *org,
	}

	mockOrgRepo.On("GetByID", orgID).Return(org, nil)
	mockUserRepo.On("GetByUsername", "invitee@example.com").Return(nil, gorm.ErrRecordNotFound)
	mockOrgRepo.On("GetPendingInviteByEmail", orgID, "invitee@example.com").Return(nil, nil)
	mockOrgRepo.On("CreateInvite", mock.AnythingOfType("*models.OrganizationInvite")).Return(nil).Run(func(args mock.Arguments) {
		inv := args.Get(0).(*models.OrganizationInvite)
		inv.ID = inviteID
	})
	mockOrgRepo.On("GetInviteByID", inviteID).Return(invite, nil)
	mockEmailService.On("SendOrganizationInvite", invite, "http://localhost:4201").Return(nil)

	result, err := service.InviteUser(orgID, req, inviterID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "invitee@example.com", result.Email)
	assert.Equal(t, models.OrganizationRoleUser, result.Role)

	mockOrgRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockEmailService.AssertExpectations(t)
}

func TestOrganizationService_InviteUser_UserAlreadyMember(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	orgID := uuid.New()
	inviterID := uuid.New()
	existingUserID := uuid.New()

	org := &models.Organization{
		ID:        orgID,
		Name:      "Test Organization",
		CreatedBy: inviterID,
	}

	existingUser := &models.User{
		ID:       existingUserID,
		Username: "existing@example.com",
	}

	existingMember := &models.OrganizationMember{
		OrganizationID: orgID,
		UserID:         existingUserID,
		Role:           models.OrganizationRoleUser,
	}

	req := &models.OrganizationInviteRequest{
		Email: "existing@example.com",
		Role:  models.OrganizationRoleUser,
	}

	mockOrgRepo.On("GetByID", orgID).Return(org, nil)
	mockUserRepo.On("GetByUsername", "existing@example.com").Return(existingUser, nil)
	mockOrgRepo.On("GetMember", orgID, existingUserID).Return(existingMember, nil)

	result, err := service.InviteUser(orgID, req, inviterID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already a member")

	mockOrgRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestOrganizationService_AcceptInvite_Success(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	userID := uuid.New()
	orgID := uuid.New()
	token := "valid-token"

	user := &models.User{
		ID:       userID,
		Username: "user@example.com",
		Name:     "Test User",
	}

	invite := &models.OrganizationInvite{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Email:          "user@example.com",
		Role:           models.OrganizationRoleUser,
		Status:         models.InviteStatusPending,
		Token:          token,
		ExpiresAt:      time.Now().Add(time.Hour * 24),
	}

	member := &models.OrganizationMember{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           models.OrganizationRoleUser,
	}

	mockOrgRepo.On("GetInviteByToken", token).Return(invite, nil)
	mockUserRepo.On("GetByID", userID).Return(user, nil)
	mockOrgRepo.On("GetMember", orgID, userID).Return(nil, nil).Once()
	mockOrgRepo.On("AddMember", mock.AnythingOfType("*models.OrganizationMember")).Return(nil)
	mockOrgRepo.On("UpdateInvite", mock.AnythingOfType("*models.OrganizationInvite")).Return(nil)
	mockOrgRepo.On("GetMember", orgID, userID).Return(member, nil).Once()

	result, err := service.AcceptInvite(token, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, orgID, result.OrganizationID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, models.OrganizationRoleUser, result.Role)

	mockOrgRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestOrganizationService_AcceptInvite_EmailMismatch(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService)

	userID := uuid.New()
	token := "valid-token"

	user := &models.User{
		ID:       userID,
		Username: "user@example.com",
	}

	invite := &models.OrganizationInvite{
		Email:     "different@example.com",
		Status:    models.InviteStatusPending,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour * 24),
	}

	mockOrgRepo.On("GetInviteByToken", token).Return(invite, nil)
	mockUserRepo.On("GetByID", userID).Return(user, nil)

	result, err := service.AcceptInvite(token, userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "email does not match")

	mockOrgRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestOrganizationService_IsUserAdmin(t *testing.T) {
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockEmailService := &MockEmailService{}
	service := NewOrganizationService(mockOrgRepo, mockUserRepo, mockEmailService).(*OrganizationService)

	orgID := uuid.New()
	creatorID := uuid.New()
	adminID := uuid.New()
	userID := uuid.New()

	org := &models.Organization{
		ID:        orgID,
		CreatedBy: creatorID,
	}

	adminMember := &models.OrganizationMember{
		OrganizationID: orgID,
		UserID:         adminID,
		Role:           models.OrganizationRoleAdmin,
	}

	regularMember := &models.OrganizationMember{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           models.OrganizationRoleUser,
	}

	// Test creator is admin
	result := service.isUserAdmin(org, creatorID)
	assert.True(t, result)

	// Test admin member
	mockOrgRepo.On("GetMember", orgID, adminID).Return(adminMember, nil)
	result = service.isUserAdmin(org, adminID)
	assert.True(t, result)

	// Test regular member
	mockOrgRepo.On("GetMember", orgID, userID).Return(regularMember, nil)
	result = service.isUserAdmin(org, userID)
	assert.False(t, result)

	// Test non-member
	nonMemberID := uuid.New()
	mockOrgRepo.On("GetMember", orgID, nonMemberID).Return(nil, gorm.ErrRecordNotFound)
	result = service.isUserAdmin(org, nonMemberID)
	assert.False(t, result)

	mockOrgRepo.AssertExpectations(t)
}
