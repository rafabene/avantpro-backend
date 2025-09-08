package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rafabene/avantpro-backend/internal/config"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/repositories"
)

// AuthService interface defines authentication and authorization operations.
// This interface provides methods for user authentication, registration,
// and password management with JWT token generation.
type AuthService interface {
	// Login authenticates a user with email and password.
	// This method validates credentials, checks for account lockout, and returns a JWT token for authorized access.
	// Parameters:
	//   - req: Login request containing email and password
	//   - ipAddress: Client IP address for security tracking
	//   - userAgent: Client user agent for security tracking
	// Returns:
	//   - *LoginResponse: JWT token and user information
	//   - error: Error if credentials are invalid, account is locked, or authentication fails
	LoginWithContext(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error)

	// Register creates a new user account and automatically logs them in.
	// This method validates user data, creates the account, and returns a JWT token.
	// Parameters:
	//   - req: Registration request containing email, name, and password
	// Returns:
	//   - *LoginResponse: JWT token and user information
	//   - error: Error if validation fails or user already exists
	Register(req *RegisterRequest) (*LoginResponse, error)

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
// registration, password management, JWT token generation and account security.
type authService struct {
	userRepo          repositories.UserRepository          // Repository for user data operations
	passwordResetRepo repositories.PasswordResetRepository // Repository for password reset token operations
	jwtSecret         string                               // Secret key for JWT token signing and validation
	authConfig        *config.AuthConfig                   // Authentication configuration
	jwtConfig         *config.JWTConfig                    // JWT configuration for token expiration
	validator         *validator.Validate                  // Validator for input validation
}

// NewAuthService creates a new AuthService instance.
// This constructor initializes the authentication service with required dependencies.
// Parameters:
//   - userRepo: Repository interface for user data operations
//   - passwordResetRepo: Repository interface for password reset token operations
//   - jwtSecret: Secret key for JWT token signing (should be strong and secure)
//   - authConfig: Authentication configuration for security settings
//   - jwtConfig: JWT configuration for token expiration settings
//
// Returns:
//   - AuthService: Configured authentication service ready for use
func NewAuthService(userRepo repositories.UserRepository, passwordResetRepo repositories.PasswordResetRepository, jwtSecret string, authConfig *config.AuthConfig, jwtConfig *config.JWTConfig) AuthService {
	return &authService{
		userRepo:          userRepo,
		passwordResetRepo: passwordResetRepo,
		jwtSecret:         jwtSecret,
		authConfig:        authConfig,
		jwtConfig:         jwtConfig,
		validator:         validator.New(),
	}
}

// LoginWithContext authenticates a user with enhanced security features
func (s *authService) LoginWithContext(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Get user from database
	user, err := s.userRepo.GetByUsername(req.Email)
	if err != nil {
		log.Printf("Tentativa de login para usuário inexistente: %s do IP: %s", req.Email, ipAddress)
		return nil, ErrInvalidCredentials
	}

	// Check if account is locked
	if user.IsLocked() {
		remaining := user.GetRemainingLockTime()
		log.Printf("Tentativa de login bloqueada para usuário bloqueado: %s do IP: %s, tempo restante de bloqueio: %v", user.Username, ipAddress, remaining.Truncate(time.Second))
		return nil, fmt.Errorf("%w: Conta bloqueada devido a muitas tentativas de login falhadas. Tente novamente em %v", ErrAccountLocked, remaining.Truncate(time.Second))
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		// Record failed attempt
		user.RecordFailedLogin()
		log.Printf("Tentativa de login falhou %d/%d para usuário: %s do IP: %s", user.FailedLoginAttempts, s.authConfig.MaxLoginAttempts, user.Username, ipAddress)

		// Check if we need to lock the account
		if user.FailedLoginAttempts >= s.authConfig.MaxLoginAttempts {
			user.LockAccount(s.authConfig.AccountLockoutDuration)
			log.Printf("Conta bloqueada para usuário %s após %d tentativas falhadas do IP: %s", user.Username, user.FailedLoginAttempts, ipAddress)
		}

		// Update user with failed attempt info
		if err := s.userRepo.Update(user); err != nil {
			log.Printf("Erro ao atualizar usuário após login falhado: %v", err)
		}

		return nil, ErrInvalidCredentials
	}

	// Clear failed attempts on successful login
	user.ResetFailedAttempts()
	if err := s.userRepo.Update(user); err != nil {
		log.Printf("Erro ao limpar tentativas falhadas para %s: %v", user.Username, err)
	}

	// Generate JWT token
	token, err := s.generateJWT(user.ID)
	if err != nil {
		return nil, err
	}

	userResponse := UserResponse{
		ID:                         user.ID,
		Username:                   user.Username,
		Name:                       user.Name,
		LastSelectedOrganizationID: user.LastSelectedOrganizationID,
		CreatedAt:                  user.CreatedAt,
		UpdatedAt:                  user.UpdatedAt,
	}

	if user.Profile != nil {
		userResponse.Profile = &ProfileResponse{
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

	return &LoginResponse{
		Token: token,
		User:  userResponse,
	}, nil
}

// Register creates a new user account and returns a JWT token
func (s *authService) Register(req *RegisterRequest) (*LoginResponse, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("dados inválidos: %w", err)
	}

	// Check if user already exists
	existingUser, _ := s.userRepo.GetByUsername(req.Email)
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Create user
	user := &models.User{
		Username: req.Email,
		Name:     req.Name,
		Password: req.Password,
	}

	// Convert to models.User for repository
	repoUser := &models.User{
		Username: user.Username,
		Name:     user.Name,
		Password: user.Password,
	}

	err := s.userRepo.Create(repoUser)
	if err != nil {
		return nil, err
	}

	// Copy back generated fields
	user.ID = repoUser.ID
	user.CreatedAt = repoUser.CreatedAt
	user.UpdatedAt = repoUser.UpdatedAt

	// Generate token
	token, err := s.generateJWT(user.ID)
	if err != nil {
		return nil, err
	}

	userResponse := UserResponse{
		ID:                         user.ID,
		Username:                   user.Username,
		Name:                       user.Name,
		LastSelectedOrganizationID: user.LastSelectedOrganizationID,
		CreatedAt:                  user.CreatedAt,
		UpdatedAt:                  user.UpdatedAt,
	}

	return &LoginResponse{
		Token: token,
		User:  userResponse,
	}, nil
}

// RequestPasswordReset sends a password reset token
func (s *authService) RequestPasswordReset(email string) error {
	user, err := s.userRepo.GetByUsername(email)
	if err != nil {
		// Por motivos de segurança, sempre retorna sucesso mesmo quando o usuário não existe
		// Isso previne ataques de enumeração de usuários
		log.Printf("Tentativa de reset de senha para email inexistente: %s", email)
		return nil
	}

	// Delete any existing tokens for this user
	if err := s.passwordResetRepo.DeleteUserTokens(user.ID); err != nil {
		log.Printf("Erro ao deletar tokens existentes para usuário %s: %v", user.ID, err)
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
		log.Printf("Erro ao criar token de reset de senha: %v", err)
		return err
	}
	log.Printf("Token de reset de senha criado com sucesso no banco de dados")

	// Log the password reset request (simulating email sending)
	log.Printf("\n\n====== SOLICITAÇÃO DE RESET DE SENHA ======")
	log.Printf("Email do Usuário: %s", email)
	log.Printf("Token de Reset: %s", resetToken)
	log.Printf("URL de Reset: http://localhost:4200/auth/password-reset/confirm?token=%s", resetToken)
	log.Printf("Token expira em: %s", tokenRecord.ExpiresAt.Format("2006-01-02 15:04:05"))
	log.Printf("==========================================\n")

	return nil
}

// ResetPassword resets user password using a token
func (s *authService) ResetPassword(token, newPassword string) error {
	if token == "" {
		return ErrTokenInvalidOrExpired
	}

	// Validate new password using a temporary struct
	resetReq := struct {
		NewPassword string `validate:"required,min=8,containsany=ABCDEFGHIJKLMNOPQRSTUVWXYZ,containsany=abcdefghijklmnopqrstuvwxyz,containsany=0123456789,containsany=!@#$%^&*()"`
	}{
		NewPassword: newPassword,
	}

	if err := s.validator.Struct(&resetReq); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPassword, err)
	}

	// Get the token from database
	log.Printf("Buscando token de reset: %s", token)
	resetToken, err := s.passwordResetRepo.GetByToken(token)
	if err != nil {
		log.Printf("Erro ao buscar token de reset: %v", err)
		return ErrTokenInvalidOrExpired
	}
	if resetToken == nil {
		log.Printf("Token de reset não encontrado no banco de dados")
		return ErrTokenInvalidOrExpired
	}
	log.Printf("Token de reset encontrado, UserID: %s, ExpiresAt: %s", resetToken.UserID, resetToken.ExpiresAt)

	// Validate the token
	if !resetToken.IsValid() {
		if resetToken.IsExpired() {
			return ErrTokenInvalidOrExpired
		}
		if resetToken.IsUsed() {
			return ErrTokenInvalidOrExpired
		}
		return ErrTokenInvalidOrExpired
	}

	// Get the user
	user, err := s.userRepo.GetByID(resetToken.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Update the user's password
	user.Password = newPassword
	// Manually hash the password before updating
	if err := user.HashPassword(); err != nil {
		log.Printf("Erro ao fazer hash da nova senha: %v", err)
		return err
	}
	log.Printf("Hash da senha realizado com sucesso para usuário: %s", user.Username)

	// Use a more specific update to avoid INSERT behavior
	if err := s.userRepo.UpdatePassword(user.ID, user.Password); err != nil {
		log.Printf("Erro ao atualizar senha do usuário: %v", err)
		return err
	}

	// Mark the token as used
	resetToken.MarkAsUsed()
	if err := s.passwordResetRepo.Update(resetToken); err != nil {
		log.Printf("Erro ao marcar token como usado: %v", err)
		// Don't fail the operation if we can't mark the token as used
	}

	log.Printf("Senha redefinida com sucesso para usuário: %s", user.Username)
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

// generateJWT creates a JWT token for the user with configurable expiration
func (s *authService) generateJWT(userID uuid.UUID) (string, error) {
	now := time.Now()
	expirationDuration := time.Duration(s.jwtConfig.ExpirationHours) * time.Hour

	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     now.Add(expirationDuration).Unix(), // Configurável via JWT_EXPIRATION_HOURS
		"iat":     now.Unix(),
		"nbf":     now.Unix(), // Not valid before current time
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
