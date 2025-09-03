package repositories

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// PasswordResetRepository interface defines password reset token operations
type PasswordResetRepository interface {
	// Create creates a new password reset token
	Create(token *models.PasswordResetToken) error

	// GetByToken retrieves a password reset token by its token string
	GetByToken(token string) (*models.PasswordResetToken, error)

	// Update updates an existing password reset token
	Update(token *models.PasswordResetToken) error

	// DeleteExpiredTokens removes all expired tokens from database
	DeleteExpiredTokens() error

	// DeleteUserTokens removes all tokens for a specific user
	DeleteUserTokens(userID uuid.UUID) error
}

// passwordResetRepository implements PasswordResetRepository interface
type passwordResetRepository struct {
	db *gorm.DB
}

// NewPasswordResetRepository creates a new password reset repository instance
func NewPasswordResetRepository(db *gorm.DB) PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

// Create creates a new password reset token
func (r *passwordResetRepository) Create(token *models.PasswordResetToken) error {
	return r.db.Create(token).Error
}

// GetByToken retrieves a password reset token by its token string
func (r *passwordResetRepository) GetByToken(token string) (*models.PasswordResetToken, error) {
	var resetToken models.PasswordResetToken

	err := r.db.Preload("User").Where("token = ? AND deleted_at IS NULL", token).First(&resetToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &resetToken, nil
}

// Update updates an existing password reset token
func (r *passwordResetRepository) Update(token *models.PasswordResetToken) error {
	return r.db.Save(token).Error
}

// DeleteExpiredTokens removes all expired tokens from database
func (r *passwordResetRepository) DeleteExpiredTokens() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&models.PasswordResetToken{}).Error
}

// DeleteUserTokens removes all tokens for a specific user
func (r *passwordResetRepository) DeleteUserTokens(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.PasswordResetToken{}).Error
}
