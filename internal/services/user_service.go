package services

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	customErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/repositories"
)

// UserService defines the interface for user business logic
type UserService interface {
	// CreateUser validates and creates a new user
	CreateUser(req *models.UserCreateRequest) (*models.UserResponse, error)
	
	// GetUserByID retrieves a user by their unique identifier
	GetUserByID(id uuid.UUID) (*models.UserResponse, error)
	
	// GetUserByUsername retrieves a user by their username (email)
	GetUserByUsername(username string) (*models.UserResponse, error)
	
	// ListUsers retrieves a paginated list with validation and sorting
	ListUsers(page, limit int, sortBy, sortOrder string) (*models.UserListResponse, error)
	
	// UpdateUser validates and updates an existing user
	UpdateUser(id uuid.UUID, req *models.UserUpdateRequest) (*models.UserResponse, error)
	
	// DeleteUser removes a user by their unique identifier
	DeleteUser(id uuid.UUID) error
}

// userService implements the UserService interface
type userService struct {
	repo      repositories.UserRepository
	validator *validator.Validate
}

// NewUserService creates a new UserService instance
func NewUserService(repo repositories.UserRepository) UserService {
	return &userService{
		repo:      repo,
		validator: validator.New(),
	}
}

// CreateUser validates and creates a new user
func (s *userService) CreateUser(req *models.UserCreateRequest) (*models.UserResponse, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, customErrors.FormatValidationError(err)
	}

	// Check if username already exists
	existingUser, err := s.repo.GetByUsername(req.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New(customErrors.ErrUsernameAlreadyExists)
	}

	// Create user entity
	user := &models.User{
		Username: strings.ToLower(strings.TrimSpace(req.Username)),
		Name:     strings.TrimSpace(req.Name),
		Password: req.Password,
	}

	// Create profile if provided
	if req.Profile != nil {
		if err := s.validator.Struct(req.Profile); err != nil {
			return nil, customErrors.FormatValidationError(err)
		}

		user.Profile = &models.Profile{
			Street:   strings.TrimSpace(req.Profile.Street),
			City:     strings.TrimSpace(req.Profile.City),
			District: strings.TrimSpace(req.Profile.District),
			ZipCode:  strings.ReplaceAll(req.Profile.ZipCode, "-", ""), // Remove dash from zip code
			Phone:    strings.ReplaceAll(strings.ReplaceAll(req.Profile.Phone, "(", ""), ")", ""), // Remove parentheses from phone
		}
		// Remove spaces and dashes from phone
		user.Profile.Phone = strings.ReplaceAll(user.Profile.Phone, " ", "")
		user.Profile.Phone = strings.ReplaceAll(user.Profile.Phone, "-", "")
	}

	// Save to repository
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return s.toUserResponse(user), nil
}

// GetUserByID retrieves a user by their unique identifier
func (s *userService) GetUserByID(id uuid.UUID) (*models.UserResponse, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(customErrors.ErrUserNotFound)
		}
		return nil, err
	}

	return s.toUserResponse(user), nil
}

// GetUserByUsername retrieves a user by their username (email)
func (s *userService) GetUserByUsername(username string) (*models.UserResponse, error) {
	user, err := s.repo.GetByUsername(strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(customErrors.ErrUserNotFound)
		}
		return nil, err
	}

	return s.toUserResponse(user), nil
}

// ListUsers retrieves a paginated list with validation and sorting
func (s *userService) ListUsers(page, limit int, sortBy, sortOrder string) (*models.UserListResponse, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	// Normalize sort parameters
	allowedSortFields := map[string]string{
		"name":      "name",
		"username":  "username",
		"createdAt": "created_at",
		"updatedAt": "updated_at",
	}

	dbSortField, exists := allowedSortFields[sortBy]
	if !exists {
		dbSortField = "created_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get users from repository
	users, total, err := s.repo.List(limit, offset, dbSortField, sortOrder)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = *s.toUserResponse(&user)
	}

	return &models.UserListResponse{
		Data:  userResponses,
		Total: total,
		Limit: limit,
		Page:  page,
	}, nil
}

// UpdateUser validates and updates an existing user
func (s *userService) UpdateUser(id uuid.UUID, req *models.UserUpdateRequest) (*models.UserResponse, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, customErrors.FormatValidationError(err)
	}

	// Get existing user
	user, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(customErrors.ErrUserNotFound)
		}
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
	}

	if req.Password != nil {
		user.Password = *req.Password
	}

	// Update profile if provided
	if req.Profile != nil {
		if user.Profile == nil {
			user.Profile = &models.Profile{UserID: user.ID}
		}

		if req.Profile.Street != nil {
			user.Profile.Street = strings.TrimSpace(*req.Profile.Street)
		}
		if req.Profile.City != nil {
			user.Profile.City = strings.TrimSpace(*req.Profile.City)
		}
		if req.Profile.District != nil {
			user.Profile.District = strings.TrimSpace(*req.Profile.District)
		}
		if req.Profile.ZipCode != nil {
			user.Profile.ZipCode = strings.ReplaceAll(*req.Profile.ZipCode, "-", "")
		}
		if req.Profile.Phone != nil {
			phone := strings.ReplaceAll(strings.ReplaceAll(*req.Profile.Phone, "(", ""), ")", "")
			phone = strings.ReplaceAll(phone, " ", "")
			phone = strings.ReplaceAll(phone, "-", "")
			user.Profile.Phone = phone
		}
	}

	// Save to repository
	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	return s.toUserResponse(user), nil
}

// DeleteUser removes a user by their unique identifier
func (s *userService) DeleteUser(id uuid.UUID) error {
	// Check if user exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(customErrors.ErrUserNotFound)
		}
		return err
	}

	return s.repo.Delete(id)
}

// toUserResponse converts a User entity to UserResponse DTO
func (s *userService) toUserResponse(user *models.User) *models.UserResponse {
	response := &models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	if user.Profile != nil {
		response.Profile = &models.ProfileResponse{
			ID:       user.Profile.ID,
			Street:   user.Profile.Street,
			City:     user.Profile.City,
			District: user.Profile.District,
			ZipCode:  user.Profile.ZipCode,
			Phone:    user.Profile.Phone,
			CreatedAt: user.Profile.CreatedAt,
			UpdatedAt: user.Profile.UpdatedAt,
		}
	}

	return response
}