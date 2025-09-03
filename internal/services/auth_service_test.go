package services

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) List(limit, offset int, sortBy, sortOrder string) ([]models.User, int64, error) {
	args := m.Called(limit, offset, sortBy, sortOrder)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateLastSelectedOrganization(userID uuid.UUID, organizationID *uuid.UUID) error {
	args := m.Called(userID, organizationID)
	return args.Error(0)
}

// MockPasswordResetRepository is a mock implementation of PasswordResetRepository
type MockPasswordResetRepository struct {
	mock.Mock
}

func (m *MockPasswordResetRepository) Create(token *models.PasswordResetToken) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockPasswordResetRepository) GetByToken(token string) (*models.PasswordResetToken, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PasswordResetToken), args.Error(1)
}

func (m *MockPasswordResetRepository) Update(token *models.PasswordResetToken) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockPasswordResetRepository) DeleteExpiredTokens() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPasswordResetRepository) DeleteUserTokens(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func TestNewAuthService(t *testing.T) {
	mockUserRepo := &MockUserRepository{}
	mockPasswordResetRepo := &MockPasswordResetRepository{}
	secret := "test-secret"

	service := NewAuthService(mockUserRepo, mockPasswordResetRepo, secret)

	assert.NotNil(t, service)
	assert.IsType(t, &authService{}, service)
}

func TestAuthService_Login_Success(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret)

	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Username: "test@example.com",
		Name:     "Test User",
		Password: "$2a$10$encrypted.password.hash",
		Profile: &models.Profile{
			ID:       uuid.New(),
			Street:   "123 Test St",
			City:     "Test City",
			District: "Test District",
			ZipCode:  "12345678",
			Phone:    "11987654321",
		},
	}

	// Mock password check to return true
	err := user.HashPassword()
	require.NoError(t, err)

	mockRepo.On("GetByUsername", "test@example.com").Return(user, nil)

	req := &models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	// Set the password for checking
	user.Password = "password123"
	err = user.HashPassword()
	require.NoError(t, err)

	result, err := service.Login(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, userID, result.User.ID)
	assert.Equal(t, "test@example.com", result.User.Username)
	assert.Equal(t, "Test User", result.User.Name)
	assert.NotNil(t, result.User.Profile)

	// Verify token is valid JWT
	token, err := jwt.Parse(result.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret)

	mockRepo.On("GetByUsername", "nonexistent@example.com").Return(nil, errors.New("user not found"))

	req := &models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	result, err := service.Login(req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "invalid credentials", err.Error())

	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret)

	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Username: "test@example.com",
		Name:     "Test User",
		Password: "correctpassword",
	}
	err := user.HashPassword()
	require.NoError(t, err)

	mockRepo.On("GetByUsername", "test@example.com").Return(user, nil)

	req := &models.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	result, err := service.Login(req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "invalid credentials", err.Error())

	mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_Success(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret)

	mockRepo.On("GetByUsername", "new@example.com").Return(nil, errors.New("user not found"))
	mockRepo.On("Create", mock.AnythingOfType("*models.User")).Return(nil).Run(func(args mock.Arguments) {
		user := args.Get(0).(*models.User)
		user.ID = uuid.New()
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
	})

	req := &models.RegisterRequest{
		Email:    "new@example.com",
		Name:     "New User",
		Password: "password123",
	}

	result, err := service.Register(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, "new@example.com", result.User.Username)
	assert.Equal(t, "New User", result.User.Name)

	mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_UserAlreadyExists(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret)

	existingUser := &models.User{
		ID:       uuid.New(),
		Username: "existing@example.com",
		Name:     "Existing User",
	}

	mockRepo.On("GetByUsername", "existing@example.com").Return(existingUser, nil)

	req := &models.RegisterRequest{
		Email:    "existing@example.com",
		Name:     "New User",
		Password: "password123",
	}

	result, err := service.Register(req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "user already exists", err.Error())

	mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_CreateError(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret)

	mockRepo.On("GetByUsername", "new@example.com").Return(nil, errors.New("user not found"))
	mockRepo.On("Create", mock.AnythingOfType("*models.User")).Return(errors.New("database error"))

	req := &models.RegisterRequest{
		Email:    "new@example.com",
		Name:     "New User",
		Password: "password123",
	}

	result, err := service.Register(req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "database error", err.Error())

	mockRepo.AssertExpectations(t)
}

func TestAuthService_RequestPasswordReset_Success(t *testing.T) {
	mockUserRepo := &MockUserRepository{}
	mockPasswordResetRepo := &MockPasswordResetRepository{}
	secret := "test-secret"
	service := NewAuthService(mockUserRepo, mockPasswordResetRepo, secret)

	user := &models.User{
		ID:       uuid.New(),
		Username: "test@example.com",
		Name:     "Test User",
	}

	mockUserRepo.On("GetByUsername", "test@example.com").Return(user, nil)
	mockPasswordResetRepo.On("DeleteUserTokens", user.ID).Return(nil)
	mockPasswordResetRepo.On("Create", mock.AnythingOfType("*models.PasswordResetToken")).Return(nil)

	err := service.RequestPasswordReset("test@example.com")

	assert.NoError(t, err)
	mockUserRepo.AssertExpectations(t)
	mockPasswordResetRepo.AssertExpectations(t)
}

func TestAuthService_RequestPasswordReset_UserNotFound(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret)

	mockRepo.On("GetByUsername", "nonexistent@example.com").Return(nil, errors.New("user not found"))

	err := service.RequestPasswordReset("nonexistent@example.com")

	assert.Error(t, err)
	assert.Equal(t, "user not found", err.Error())
	mockRepo.AssertExpectations(t)
}

func TestAuthService_ResetPassword_Success(t *testing.T) {
	mockUserRepo := &MockUserRepository{}
	mockPasswordResetRepo := &MockPasswordResetRepository{}
	secret := "test-secret"
	service := NewAuthService(mockUserRepo, mockPasswordResetRepo, secret)

	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Username: "test@example.com",
		Name:     "Test User",
		Password: "oldpassword",
	}

	validToken := &models.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	mockPasswordResetRepo.On("GetByToken", "valid-token").Return(validToken, nil)
	mockUserRepo.On("GetByID", userID).Return(user, nil)
	mockUserRepo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)
	mockPasswordResetRepo.On("Update", mock.AnythingOfType("*models.PasswordResetToken")).Return(nil)

	err := service.ResetPassword("valid-token", "newpassword123")

	assert.NoError(t, err)
	mockUserRepo.AssertExpectations(t)
	mockPasswordResetRepo.AssertExpectations(t)
}

func TestAuthService_ResetPassword_EmptyToken(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret)

	err := service.ResetPassword("", "newpassword123")

	assert.Error(t, err)
	assert.Equal(t, "invalid or expired token", err.Error())
}

func TestAuthService_GenerateJWT(t *testing.T) {
	mockRepo := &MockUserRepository{}
	secret := "test-secret"
	service := NewAuthService(mockRepo, &MockPasswordResetRepository{}, secret).(*authService)

	userID := uuid.New()

	token, err := service.generateJWT(userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token is valid
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	// Verify claims
	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
		assert.Equal(t, userID.String(), claims["user_id"])
		assert.NotNil(t, claims["exp"])
		assert.NotNil(t, claims["iat"])
	}
}
