package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a user entity
// @Description User information
type User struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null" validate:"required,email" example:"user@example.com"`
	Name      string         `json:"name" gorm:"not null" validate:"required,min=2,max=100" example:"John Doe"`
	Password  string         `json:"-" gorm:"not null" validate:"required,min=6"`
	Profile   *Profile       `json:"profile,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt time.Time      `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time      `json:"updated_at" example:"2023-01-01T12:00:00Z"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Profile represents a user profile with address information
// @Description User profile with address and contact information
type Profile struct {
	ID       uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440001"`
	UserID   uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;uniqueIndex" example:"550e8400-e29b-41d4-a716-446655440000"`
	Street   string         `json:"street" gorm:"not null" validate:"required,min=5,max=200" example:"123 Main Street"`
	City     string         `json:"city" gorm:"not null" validate:"required,min=2,max=100" example:"São Paulo"`
	District string         `json:"district" gorm:"not null" validate:"required,min=2,max=100" example:"Centro"`
	ZipCode  string         `json:"zip_code" gorm:"not null" validate:"required,len=8" example:"01234567"`
	Phone    string         `json:"phone" gorm:"not null" validate:"required,min=10,max=15" example:"11987654321"`
	CreatedAt time.Time     `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time     `json:"updated_at" example:"2023-01-01T12:00:00Z"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// UserCreateRequest represents the request body for creating a user
// @Description User creation request
type UserCreateRequest struct {
	Username string                `json:"username" validate:"required,email" example:"user@example.com"`
	Name     string                `json:"name" validate:"required,min=2,max=100" example:"John Doe"`
	Password string                `json:"password" validate:"required,min=6" example:"password123"`
	Profile  *ProfileCreateRequest `json:"profile,omitempty"`
}

// ProfileCreateRequest represents the request body for creating a profile
// @Description Profile creation request
type ProfileCreateRequest struct {
	Street   string `json:"street" validate:"required,min=5,max=200" example:"123 Main Street"`
	City     string `json:"city" validate:"required,min=2,max=100" example:"São Paulo"`
	District string `json:"district" validate:"required,min=2,max=100" example:"Centro"`
	ZipCode  string `json:"zip_code" validate:"required,len=8" example:"01234567"`
	Phone    string `json:"phone" validate:"required,min=10,max=15" example:"11987654321"`
}

// UserUpdateRequest represents the request body for updating a user
// @Description User update request
type UserUpdateRequest struct {
	Name     *string                `json:"name,omitempty" validate:"omitempty,min=2,max=100" example:"John Doe"`
	Password *string                `json:"password,omitempty" validate:"omitempty,min=6" example:"newpassword123"`
	Profile  *ProfileUpdateRequest  `json:"profile,omitempty"`
}

// ProfileUpdateRequest represents the request body for updating a profile
// @Description Profile update request
type ProfileUpdateRequest struct {
	Street   *string `json:"street,omitempty" validate:"omitempty,min=5,max=200" example:"123 Main Street"`
	City     *string `json:"city,omitempty" validate:"omitempty,min=2,max=100" example:"São Paulo"`
	District *string `json:"district,omitempty" validate:"omitempty,min=2,max=100" example:"Centro"`
	ZipCode  *string `json:"zip_code,omitempty" validate:"omitempty,len=8" example:"01234567"`
	Phone    *string `json:"phone,omitempty" validate:"omitempty,min=10,max=15" example:"11987654321"`
}

// UserResponse represents the response body for user operations
// @Description User response
type UserResponse struct {
	ID        uuid.UUID        `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string           `json:"username" example:"user@example.com"`
	Name      string           `json:"name" example:"John Doe"`
	Profile   *ProfileResponse `json:"profile,omitempty"`
	CreatedAt time.Time        `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time        `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// ProfileResponse represents the response body for profile operations
// @Description Profile response
type ProfileResponse struct {
	ID       uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Street   string    `json:"street" example:"123 Main Street"`
	City     string    `json:"city" example:"São Paulo"`
	District string    `json:"district" example:"Centro"`
	ZipCode  string    `json:"zip_code" example:"01234567"`
	Phone    string    `json:"phone" example:"11987654321"`
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// UserListResponse represents the response body for listing users
// @Description User list response with pagination
type UserListResponse struct {
	Data  []UserResponse `json:"data"`
	Total int64          `json:"total" example:"100"`
	Limit int            `json:"limit" example:"50"`
	Page  int            `json:"page" example:"1"`
}

// HashPassword hashes the user password using bcrypt
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifies if the provided password matches the hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// BeforeCreate is a GORM hook that runs before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	return u.HashPassword()
}

// BeforeUpdate is a GORM hook that runs before updating a user
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// Only hash password if it's being updated
	if tx.Statement.Changed("Password") {
		return u.HashPassword()
	}
	return nil
}

// LoginRequest represents the request body for user login
// @Description User login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"password123"`
}

// RegisterRequest represents the request body for user registration
// @Description User registration request
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Name     string `json:"name" validate:"required,min=2,max=100" example:"John Doe"`
	Password string `json:"password" validate:"required,min=6" example:"password123"`
}

// LoginResponse represents the response body for successful login
// @Description Login response with token and user info
type LoginResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  UserResponse `json:"user"`
}

// PasswordResetRequest represents the request body for password reset
// @Description Password reset request
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

// PasswordResetConfirmRequest represents the request body for password reset confirmation
// @Description Password reset confirmation request
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" validate:"required" example:"reset-token-123"`
	NewPassword string `json:"new_password" validate:"required,min=6" example:"newpassword123"`
}

// MessageResponse represents a simple message response
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}