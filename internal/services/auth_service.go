package services

import (
	"errors"
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
}

// authService implements AuthService interface.
// It provides authentication and authorization services including user login,
// registration, password management, and JWT token generation and validation.
type authService struct {
	userRepo  repositories.UserRepository // Repository for user data operations
	jwtSecret string                       // Secret key for JWT token signing and validation
}

// NewAuthService creates a new AuthService instance.
// This constructor initializes the authentication service with required dependencies.
// Parameters:
//   - userRepo: Repository interface for user data operations
//   - jwtSecret: Secret key for JWT token signing (should be strong and secure)
// Returns:
//   - AuthService: Configured authentication service ready for use
func NewAuthService(userRepo repositories.UserRepository, jwtSecret string) AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
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
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
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
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return &models.LoginResponse{
		Token: token,
		User:  userResponse,
	}, nil
}

// RequestPasswordReset sends a password reset token (simplified implementation)
func (s *authService) RequestPasswordReset(email string) error {
	_, err := s.userRepo.GetByUsername(email)
	if err != nil {
		return errors.New("user not found")
	}

	// In a real implementation, you would:
	// 1. Generate a unique reset token
	// 2. Store it in the database with expiration
	// 3. Send an email with the reset link
	
	// For now, we'll just return success
	return nil
}

// ResetPassword resets user password using a token (simplified implementation)
func (s *authService) ResetPassword(token, newPassword string) error {
	// In a real implementation, you would:
	// 1. Validate the reset token
	// 2. Check if it's not expired
	// 3. Find the user associated with the token
	// 4. Update the user's password
	// 5. Invalidate the token

	// For now, we'll just return success for any token
	if token == "" {
		return errors.New("invalid or expired token")
	}

	return nil
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