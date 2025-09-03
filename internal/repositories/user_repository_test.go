package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/models"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Skip("Docker not available, skipping testcontainer tests")
		return nil, func() {}
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{})
	require.NoError(t, err)

	// Enable UUID extension
	err = db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error
	require.NoError(t, err)

	// Auto migrate
	err = db.AutoMigrate(&models.User{}, &models.Profile{})
	require.NoError(t, err)

	cleanup := func() {
		_ = pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

func TestNewUserRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)
	assert.NotNil(t, repo)
}

func TestUserRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)

	user := &models.User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}

	err := repo.Create(user)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.NotEqual(t, "password123", user.Password) // Password should be hashed
}

func TestUserRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)

	// Create a user first
	user := &models.User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}
	err := repo.Create(user)
	require.NoError(t, err)

	// Test GetByID
	foundUser, err := repo.GetByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, foundUser.ID)
	assert.Equal(t, user.Username, foundUser.Username)
	assert.Equal(t, user.Name, foundUser.Name)

	// Test non-existent ID
	nonExistentID := uuid.New()
	_, err = repo.GetByID(nonExistentID)
	assert.Error(t, err)
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)

	// Create a user first
	user := &models.User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}
	err := repo.Create(user)
	require.NoError(t, err)

	// Test GetByUsername
	foundUser, err := repo.GetByUsername(user.Username)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, foundUser.ID)
	assert.Equal(t, user.Username, foundUser.Username)
	assert.Equal(t, user.Name, foundUser.Name)

	// Test non-existent username
	_, err = repo.GetByUsername("nonexistent@example.com")
	assert.Error(t, err)
}

func TestUserRepository_GetByID_WithProfile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)

	// Create a user with profile
	user := &models.User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Profile: &models.Profile{
			Street:   "123 Main Street",
			City:     "São Paulo",
			District: "Centro",
			ZipCode:  "01234567",
			Phone:    "11987654321",
		},
	}
	err := repo.Create(user)
	require.NoError(t, err)

	// Test GetByID with profile preloaded
	foundUser, err := repo.GetByID(user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundUser.Profile)
	assert.Equal(t, user.Profile.Street, foundUser.Profile.Street)
	assert.Equal(t, user.Profile.City, foundUser.Profile.City)
}

func TestUserRepository_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)

	// Create multiple users
	users := []*models.User{
		{Username: "user1@example.com", Name: "User One", Password: "password123"},
		{Username: "user2@example.com", Name: "User Two", Password: "password123"},
		{Username: "user3@example.com", Name: "User Three", Password: "password123"},
	}

	for _, user := range users {
		err := repo.Create(user)
		require.NoError(t, err)
	}

	// Test list without sorting
	foundUsers, total, err := repo.List(10, 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, foundUsers, 3)

	// Test list with limit
	foundUsers, total, err = repo.List(2, 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, foundUsers, 2)

	// Test list with offset
	foundUsers, total, err = repo.List(10, 1, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, foundUsers, 2)

	// Test list with valid sorting
	foundUsers, total, err = repo.List(10, 0, "name", "asc")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, foundUsers, 3)
	assert.Equal(t, "User One", foundUsers[0].Name)

	// Test list with invalid sorting (should use default)
	foundUsers, total, err = repo.List(10, 0, "invalid_field", "asc")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, foundUsers, 3)
}

func TestUserRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)

	// Create a user first
	user := &models.User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}
	err := repo.Create(user)
	require.NoError(t, err)

	// Update the user
	user.Name = "Updated User"
	err = repo.Update(user)
	assert.NoError(t, err)

	// Verify the update
	foundUser, err := repo.GetByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated User", foundUser.Name)
}

func TestUserRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)

	// Create a user first
	user := &models.User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}
	err := repo.Create(user)
	require.NoError(t, err)

	// Delete the user
	err = repo.Delete(user.ID)
	assert.NoError(t, err)

	// Verify the user is soft deleted
	_, err = repo.GetByID(user.ID)
	assert.Error(t, err) // Should not find the user because it's soft deleted
}

func TestUserRepository_SortingValidation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewUserRepository(db)

	// Create a user for testing
	user := &models.User{
		Username: "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}
	err := repo.Create(user)
	require.NoError(t, err)

	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		expectErr bool
	}{
		{"Valid name asc", "name", "asc", false},
		{"Valid name desc", "name", "desc", false},
		{"Valid username asc", "username", "asc", false},
		{"Valid created_at desc", "created_at", "desc", false},
		{"Valid updated_at asc", "updated_at", "asc", false},
		{"Invalid field", "invalid_field", "asc", false}, // Should not error, but use default sorting
		{"Invalid order", "name", "invalid", false},      // Should not error, but use default sorting
		{"Empty sort", "", "", false},                    // Should use default sorting
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, total, err := repo.List(10, 0, tt.sortBy, tt.sortOrder)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(1), total)
				assert.Len(t, users, 1)
			}
		})
	}
}
