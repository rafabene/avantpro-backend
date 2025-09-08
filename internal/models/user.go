package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User representa uma entidade de usuário
type User struct {
	ID                         uuid.UUID  `gorm:"type:char(36);primaryKey"`
	Username                   string     `gorm:"uniqueIndex;not null"`
	Name                       string     `gorm:"not null"`
	Password                   string     `gorm:"not null"`
	LastSelectedOrganizationID *uuid.UUID `gorm:"type:char(36)"`
	FailedLoginAttempts        int        `gorm:"default:0"`
	LastFailedLoginAt          *time.Time
	LockedUntil                *time.Time
	Profile                    *Profile `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	DeletedAt                  gorm.DeletedAt `gorm:"index"`
}

// Profile representa o perfil de um usuário com informações de endereço
type Profile struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex"`
	Street    string    `gorm:"not null"`
	City      string    `gorm:"not null"`
	District  string    `gorm:"not null"`
	ZipCode   string    `gorm:"not null"`
	Phone     string    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// HashPassword faz o hash da senha do usuário usando bcrypt
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifica se a senha fornecida corresponde à senha com hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// BeforeCreate é um hook do GORM que executa antes de criar um usuário
func (u *User) BeforeCreate(tx *gorm.DB) error {
	return u.HashPassword()
}

// BeforeUpdate é um hook do GORM que executa antes de atualizar um usuário
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// Fazer hash da senha apenas se estiver sendo atualizada
	if tx.Statement.Changed("Password") {
		return u.HashPassword()
	}
	return nil
}

// IsLocked verifica se a conta do usuário está atualmente bloqueada
func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && time.Now().Before(*u.LockedUntil)
}

// GetRemainingLockTime retorna o tempo restante até o desbloqueio
func (u *User) GetRemainingLockTime() time.Duration {
	if !u.IsLocked() {
		return 0
	}
	return time.Until(*u.LockedUntil)
}

// RecordFailedLogin incrementa as tentativas de login falhadas e atualiza o timestamp
func (u *User) RecordFailedLogin() {
	u.FailedLoginAttempts++
	now := time.Now()
	u.LastFailedLoginAt = &now
}

// LockAccount bloqueia a conta pela duração especificada
func (u *User) LockAccount(duration time.Duration) {
	lockUntil := time.Now().Add(duration)
	u.LockedUntil = &lockUntil
}

// UnlockAccount desbloqueia a conta e redefine as tentativas falhadas
func (u *User) UnlockAccount() {
	u.LockedUntil = nil
	u.FailedLoginAttempts = 0
	u.LastFailedLoginAt = nil
}

// ResetFailedAttempts limpa as tentativas de login falhadas após login bem-sucedido
func (u *User) ResetFailedAttempts() {
	u.FailedLoginAttempts = 0
	u.LastFailedLoginAt = nil
}
