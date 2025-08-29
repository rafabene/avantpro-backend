package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Create inserts a new user into the database
	Create(user *models.User) error
	
	// GetByID retrieves a user by their unique identifier
	GetByID(id uuid.UUID) (*models.User, error)
	
	// GetByUsername retrieves a user by their username (email)
	GetByUsername(username string) (*models.User, error)
	
	// List retrieves a paginated list of users with total count and sorting
	List(limit, offset int, sortBy, sortOrder string) ([]models.User, int64, error)
	
	// Update modifies an existing user in the database
	Update(user *models.User) error
	
	// Delete removes a user from the database (soft delete)
	Delete(id uuid.UUID) error
}

// userRepository implements the UserRepository interface
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create inserts a new user into the database
func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByID retrieves a user by their unique identifier
func (r *userRepository) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Profile").Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by their username (email)
func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Profile").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// List retrieves a paginated list of users with total count and sorting
func (r *userRepository) List(limit, offset int, sortBy, sortOrder string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Count total records
	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build base query
	query := r.db.Preload("Profile").Limit(limit).Offset(offset)

	// Apply sorting with validation
	allowedFields := map[string]bool{
		"name":       true,
		"username":   true,
		"created_at": true,
		"updated_at": true,
	}

	if allowedFields[sortBy] && (sortOrder == "asc" || sortOrder == "desc") {
		query = query.Order(sortBy + " " + sortOrder)
	} else {
		// Default sorting
		query = query.Order("created_at DESC")
	}

	// Execute query
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Update modifies an existing user in the database
func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Delete removes a user from the database (soft delete)
func (r *userRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, id).Error
}