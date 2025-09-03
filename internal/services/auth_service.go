package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/repositories"
)

// AuthService interface defines authentication and authorization operations.
// This interface provides methods for user authentication, registration,
// and password management with JWT token generation.
type AuthService interface {
	// Login authenticates a user with email and password.
	// This method validates credentials and returns a JWT token for authorized access.
	// Parameters:
	//   - req: Login request containing email and password
	// Returns:
	//   - *models.LoginResponse: JWT token and user information
	//   - error: Error if credentials are invalid or authentication fails
	Login(req *models.LoginRequest) (*models.LoginResponse, error)

	// Register creates a new user account and automatically logs them in.
	// This method validates user data, creates the account, and returns a JWT token.
	// Parameters:
	//   - req: Registration request containing email, name, and password
	// Returns:
	//   - *models.LoginResponse: JWT token and user information
	//   - error: Error if validation fails or user already exists
	Register(req *models.RegisterRequest) (*models.LoginResponse, error)

	// RequestPasswordReset initiates the password reset process for a user.
	// This method generates a reset token and sends a password reset email.
	// Parameters:
	//   - email: Email address of the user requesting password reset
	// Returns:
	//   - error: Error if user not found or email sending fails
	RequestPasswordReset(email string) error

	// ResetPassword completes the password reset process using a reset token.
	// This method validates the reset token and updates the user's password.
	// Parameters:
	//   - token: Password reset token from email
	//   - newPassword: New password to set for the user
	// Returns:
	//   - error: Error if token is invalid, expired, or password update fails
	ResetPassword(token, newPassword string) error

	// UpdateLastSelectedOrganization updates the user's last selected organization preference.
	// This method allows users to save their organization preference for automatic selection on login.
	// Parameters:
	//   - userID: UUID of the user
	//   - organizationID: UUID of the organization to set as preferred (nil to clear preference)
	// Returns:
	//   - error: Error if user not found or update fails
	UpdateLastSelectedOrganization(userID uuid.UUID, organizationID *uuid.UUID) error
}

// authService implements AuthService interface.
// It provides authentication and authorization services including user login,
// registration, password management, and JWT token generation and validation.
type authService struct {
	userRepo          repositories.UserRepository          // Repository for user data operations
	passwordResetRepo repositories.PasswordResetRepository // Repository for password reset token operations
	jwtSecret         string                               // Secret key for JWT token signing and validation
}

// NewAuthService creates a new AuthService instance.
// This constructor initializes the authentication service with required dependencies.
// Parameters:
//   - userRepo: Repository interface for user data operations
//   - passwordResetRepo: Repository interface for password reset token operations
//   - jwtSecret: Secret key for JWT token signing (should be strong and secure)
//
// Returns:
//   - AuthService: Configured authentication service ready for use
func NewAuthService(userRepo repositories.UserRepository, passwordResetRepo repositories.PasswordResetRepository, jwtSecret string) AuthService {
	return &authService{
		userRepo:          userRepo,
		passwordResetRepo: passwordResetRepo,
		jwtSecret:         jwtSecret,
	}
}

// Login authenticates a user and returns a JWT token
func (s *authService) Login(req *models.LoginRequest) (*models.LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.CheckPassword(req.Password) {
		return nil, errors.New("invalid credentials")
	}

	token, err := s.generateJWT(user.ID)
	if err != nil {
		return nil, err
	}

	userResponse := models.UserResponse{
		ID:                         user.ID,
		Username:                   user.Username,
		Name:                       user.Name,
		LastSelectedOrganizationID: user.LastSelectedOrganizationID,
		CreatedAt:                  user.CreatedAt,
		UpdatedAt:                  user.UpdatedAt,
	}

	if user.Profile != nil {
		userResponse.Profile = &models.ProfileResponse{
			ID:        user.Profile.ID,
			Street:    user.Profile.Street,
			City:      user.Profile.City,
			District:  user.Profile.District,
			ZipCode:   user.Profile.ZipCode,
			Phone:     user.Profile.Phone,
			CreatedAt: user.Profile.CreatedAt,
			UpdatedAt: user.Profile.UpdatedAt,
		}
	}

	return &models.LoginResponse{
		Token: token,
		User:  userResponse,
	}, nil
}

// Register creates a new user account and returns a JWT token
func (s *authService) Register(req *models.RegisterRequest) (*models.LoginResponse, error) {
	// Check if user already exists
	existingUser, _ := s.userRepo.GetByUsername(req.Email)
	if existingUser != nil {
		return nil, errors.New("user already exists")
	}

	// Create user
	user := &models.User{
		Username: req.Email,
		Name:     req.Name,
		Password: req.Password,
	}

	err := s.userRepo.Create(user)
	if err != nil {
		return nil, err
	}

	// Generate token
	token, err := s.generateJWT(user.ID)
	if err != nil {
		return nil, err
	}

	userResponse := models.UserResponse{
		ID:                         user.ID,
		Username:                   user.Username,
		Name:                       user.Name,
		LastSelectedOrganizationID: user.LastSelectedOrganizationID,
		CreatedAt:                  user.CreatedAt,
		UpdatedAt:                  user.UpdatedAt,
	}

	return &models.LoginResponse{
		Token: token,
		User:  userResponse,
	}, nil
}

// RequestPasswordReset sends a password reset token
func (s *authService) RequestPasswordReset(email string) error {
	user, err := s.userRepo.GetByUsername(email)
	if err != nil {
		return errors.New("user not found")
	}

	// Delete any existing tokens for this user
	if err := s.passwordResetRepo.DeleteUserTokens(user.ID); err != nil {
		log.Printf("Error deleting existing tokens for user %s: %v", user.ID, err)
	}

	// Generate a unique reset token
	resetToken := s.generateResetToken()

	// Create token record with 1 hour expiration
	tokenRecord := &models.PasswordResetToken{
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Store the token in database
	if err := s.passwordResetRepo.Create(tokenRecord); err != nil {
		log.Printf("Error creating password reset token: %v", err)
		return err
	}
	log.Printf("Password reset token created successfully in database")

	// Log the password reset request (simulating email sending)
	log.Printf("\n\n====== PASSWORD RESET REQUEST ======")
	log.Printf("User Email: %s", email)
	log.Printf("Reset Token: %s", resetToken)
	log.Printf("Reset URL: http://localhost:4200/auth/password-reset/confirm?token=%s", resetToken)
	log.Printf("Token expires at: %s", tokenRecord.ExpiresAt.Format("2006-01-02 15:04:05"))
	log.Printf("====================================\n")

	return nil
}

// ResetPassword resets user password using a token
func (s *authService) ResetPassword(token, newPassword string) error {
	if token == "" {
		return errors.New("invalid or expired token")
	}

	// Get the token from database
	log.Printf("Searching for reset token: %s", token)
	resetToken, err := s.passwordResetRepo.GetByToken(token)
	if err != nil {
		log.Printf("Error getting reset token: %v", err)
		return err
	}
	if resetToken == nil {
		log.Printf("Reset token not found in database")
		return errors.New("invalid or expired token")
	}
	log.Printf("Reset token found, UserID: %s, ExpiresAt: %s", resetToken.UserID, resetToken.ExpiresAt)

	// Validate the token
	if !resetToken.IsValid() {
		if resetToken.IsExpired() {
			return errors.New("token has expired")
		}
		if resetToken.IsUsed() {
			return errors.New("token has already been used")
		}
		return errors.New("invalid token")
	}

	// Get the user
	user, err := s.userRepo.GetByID(resetToken.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Update the user's password
	user.Password = newPassword
	// Manually hash the password before updating
	if err := user.HashPassword(); err != nil {
		log.Printf("Error hashing new password: %v", err)
		return err
	}
	log.Printf("Password hashed successfully for user: %s", user.Username)

	if err := s.userRepo.Update(user); err != nil {
		log.Printf("Error updating user password: %v", err)
		return err
	}

	// Mark the token as used
	resetToken.MarkAsUsed()
	if err := s.passwordResetRepo.Update(resetToken); err != nil {
		log.Printf("Error marking token as used: %v", err)
		// Don't fail the operation if we can't mark the token as used
	}

	log.Printf("Password successfully reset for user: %s", user.Username)
	return nil
}

// UpdateLastSelectedOrganization updates the user's last selected organization preference
func (s *authService) UpdateLastSelectedOrganization(userID uuid.UUID, organizationID *uuid.UUID) error {
	return s.userRepo.UpdateLastSelectedOrganization(userID, organizationID)
}

// generateResetToken generates a secure random token for password reset
func (s *authService) generateResetToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based token if crypto/rand fails
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405") + "resettoken"))
	}
	return hex.EncodeToString(bytes)
}

// generateJWT creates a JWT token for the user
func (s *authService) generateJWT(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // 24 hours expiration
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
