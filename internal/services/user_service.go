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

// UserService defines the interface for user business logic operations.
// This interface provides methods for managing user data with proper validation,
// error handling, and business rule enforcement.
type UserService interface {
	// CreateUser validates and creates a new user in the system.
	// This method performs comprehensive validation of user data including:
	//   - Email format validation and uniqueness check
	//   - Password strength requirements
	//   - Profile data validation if provided
	// Parameters:
	//   - req: User creation request containing user details and optional profile
	// Returns:
	//   - *models.UserResponse: Created user data (password excluded)
	//   - error: Validation or creation error
	CreateUser(req *models.UserCreateRequest) (*models.UserResponse, error)
	
	// GetUserByID retrieves a user by their unique identifier.
	// This method loads the user with their profile information if available.
	// Parameters:
	//   - id: UUID of the user to retrieve
	// Returns:
	//   - *models.UserResponse: User data with profile (password excluded)
	//   - error: Error if user not found or database error
	GetUserByID(id uuid.UUID) (*models.UserResponse, error)
	
	// GetUserByUsername retrieves a user by their username (email address).
	// This method is commonly used for login and user lookup operations.
	// Parameters:
	//   - username: Email address of the user to find
	// Returns:
	//   - *models.UserResponse: User data with profile (password excluded)
	//   - error: Error if user not found or database error
	GetUserByUsername(username string) (*models.UserResponse, error)
	
	// ListUsers retrieves a paginated list of users with validation and sorting.
	// This method supports filtering, sorting, and pagination for user management.
	// Parameters:
	//   - page: Page number for pagination (1-based)
	//   - limit: Number of users per page (max 100)
	//   - sortBy: Field to sort by (name, username, created_at, updated_at)
	//   - sortOrder: Sort direction (asc, desc)
	// Returns:
	//   - *models.UserListResponse: Paginated user list with metadata
	//   - error: Validation or query error
	ListUsers(page, limit int, sortBy, sortOrder string) (*models.UserListResponse, error)
	
	// UpdateUser validates and updates an existing user's information.
	// This method allows partial updates and maintains data integrity.
	// Only provided fields are updated, others remain unchanged.
	// Parameters:
	//   - id: UUID of the user to update
	//   - req: Update request containing new user data
	// Returns:
	//   - *models.UserResponse: Updated user data (password excluded)
	//   - error: Validation or update error
	UpdateUser(id uuid.UUID, req *models.UserUpdateRequest) (*models.UserResponse, error)
	
	// DeleteUser removes a user by their unique identifier.
	// This performs a soft delete, marking the user as deleted while preserving data.
	// The user's profile is also soft deleted through cascade operations.
	// Parameters:
	//   - id: UUID of the user to delete
	// Returns:
	//   - error: Error if user not found or deletion fails
	DeleteUser(id uuid.UUID) error
}

// userService implements the UserService interface.
// It provides business logic for user management with validation, error handling,
// and proper data transformation between domain models and API responses.
type userService struct {
	repo      repositories.UserRepository // Repository for user data operations
	validator *validator.Validate         // Validator for input validation
}

// NewUserService creates a new UserService instance.
// This constructor initializes the service with a user repository and validator.
// Parameters:
//   - repo: Repository interface for user data operations
// Returns:
//   - UserService: Configured user service ready for use
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