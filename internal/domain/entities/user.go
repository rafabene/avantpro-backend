package entities

import (
	"errors"
	"time"

	"github.com/rafabene/avantpro-backend/internal/domain/valueobjects"
)

var (
	ErrInvalidUserData = errors.New("invalid user data")
)

// User representa um usuário do sistema
type User struct {
	ID           string
	Email        valueobjects.Email
	Name         string
	PasswordHash string
	Role         Role
	AvatarURL    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time // Soft delete
}

// IsAdmin verifica se o usuário é admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// HasPermission verifica se o usuário tem uma permissão
func (u *User) HasPermission(permission Permission) bool {
	return u.Role.HasPermission(permission)
}

// GetPermissions retorna todas as permissões do usuário
func (u *User) GetPermissions() []string {
	perms := u.Role.GetPermissions()
	result := make([]string, len(perms))
	for i, p := range perms {
		result[i] = string(p)
	}
	return result
}

// IsDeleted verifica se o usuário foi deletado (soft delete)
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// SoftDelete marca o usuário como deletado
func (u *User) SoftDelete() {
	now := time.Now()
	u.DeletedAt = &now
}

// Restore restaura um usuário deletado
func (u *User) Restore() {
	u.DeletedAt = nil
}

// Validate valida regras de negócio da entidade User
func (u *User) Validate() error {
	if u.Email.String() == "" {
		return errors.New("email is required")
	}

	if u.Name == "" {
		return errors.New("name is required")
	}

	if len(u.Name) < 2 {
		return errors.New("name must be at least 2 characters")
	}

	if u.Role != RoleAdmin && u.Role != RoleUser && u.Role != RoleGuest {
		return errors.New("invalid role")
	}

	return nil
}
