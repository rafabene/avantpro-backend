package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestUser_HashPassword(t *testing.T) {
	user := &User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}

	// Test password hashing
	err := user.HashPassword()
	assert.NoError(t, err)

	// Verify password is hashed
	assert.NotEqual(t, "password123", user.Password)
	assert.True(t, len(user.Password) > 20) // bcrypt hashes are long

	// Verify password can be validated
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123"))
	assert.NoError(t, err)
}

func TestUser_CheckPassword(t *testing.T) {
	user := &User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}

	// Hash the password first
	err := user.HashPassword()
	assert.NoError(t, err)

	// Test correct password
	assert.True(t, user.CheckPassword("password123"))

	// Test incorrect password
	assert.False(t, user.CheckPassword("wrongpassword"))
}

func TestProfileValidation(t *testing.T) {
	profile := Profile{
		Street:   "123 Main Street",
		City:     "São Paulo",
		District: "Centro",
		ZipCode:  "01234567",
		Phone:    "11987654321",
	}

	// Test valid profile data
	assert.NotEmpty(t, profile.Street)
	assert.NotEmpty(t, profile.City)
	assert.NotEmpty(t, profile.District)
	assert.Equal(t, 8, len(profile.ZipCode))
	assert.True(t, len(profile.Phone) >= 10)
}
