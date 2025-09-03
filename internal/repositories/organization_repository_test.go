package repositories

import (
	"context"
	"testing"
	"time"

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

func setupOrgTestDB(t *testing.T) (*gorm.DB, func()) {
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
	err = db.AutoMigrate(&models.User{}, &models.Profile{}, &models.Organization{}, &models.OrganizationMember{}, &models.OrganizationInvite{})
	require.NoError(t, err)

	cleanup := func() {
		_ = pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

func createTestUser(t *testing.T, db *gorm.DB, email, name string) *models.User {
	user := &models.User{
		Username: email,
		Name:     name,
		Password: "password123",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

func TestNewOrganizationRepository(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	assert.NotNil(t, repo)
}

func TestOrganizationRepository_Create(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}

	err := repo.Create(org)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, org.ID)

	// Verify the creator was automatically added as admin member
	member, err := repo.GetMember(org.ID, creator.ID)
	assert.NoError(t, err)
	assert.NotNil(t, member)
	assert.Equal(t, models.OrganizationRoleAdmin, member.Role)
}

func TestOrganizationRepository_GetByID(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Test GetByID
	foundOrg, err := repo.GetByID(org.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundOrg)
	assert.Equal(t, org.ID, foundOrg.ID)
	assert.Equal(t, org.Name, foundOrg.Name)
	assert.Equal(t, org.Description, foundOrg.Description)
	assert.NotNil(t, foundOrg.Creator)
	assert.Len(t, foundOrg.Members, 1) // Creator should be auto-added as member

	// Test non-existent ID
	nonExistentID := uuid.New()
	foundOrg, err = repo.GetByID(nonExistentID)
	assert.NoError(t, err)
	assert.Nil(t, foundOrg)
}

func TestOrganizationRepository_GetByCreator(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")
	otherUser := createTestUser(t, db, "other@example.com", "Other User")

	// Create organizations
	org1 := &models.Organization{Name: "Org 1", Description: "Desc 1", CreatedBy: creator.ID}
	org2 := &models.Organization{Name: "Org 2", Description: "Desc 2", CreatedBy: creator.ID}
	org3 := &models.Organization{Name: "Org 3", Description: "Desc 3", CreatedBy: otherUser.ID}

	err := repo.Create(org1)
	require.NoError(t, err)
	err = repo.Create(org2)
	require.NoError(t, err)
	err = repo.Create(org3)
	require.NoError(t, err)

	// Test GetByCreator
	orgs, total, err := repo.GetByCreator(creator.ID, 10, 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, orgs, 2)

	// Test with pagination
	orgs, total, err = repo.GetByCreator(creator.ID, 1, 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, orgs, 1)

	// Test with sorting
	orgs, total, err = repo.GetByCreator(creator.ID, 10, 0, "name", "asc")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, orgs, 2)
	assert.Equal(t, "Org 1", orgs[0].Name)
}

func TestOrganizationRepository_Update(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Update organization
	org.Name = "Updated Organization"
	org.Description = "Updated Description"
	err = repo.Update(org)
	assert.NoError(t, err)

	// Verify update
	foundOrg, err := repo.GetByID(org.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Organization", foundOrg.Name)
	assert.Equal(t, "Updated Description", foundOrg.Description)
}

func TestOrganizationRepository_Delete(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Delete organization
	err = repo.Delete(org.ID)
	assert.NoError(t, err)

	// Verify soft delete
	foundOrg, err := repo.GetByID(org.ID)
	assert.NoError(t, err)
	assert.Nil(t, foundOrg) // Should not find due to soft delete
}

func TestOrganizationRepository_List(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create multiple organizations
	orgs := []*models.Organization{
		{Name: "Org A", Description: "Desc A", CreatedBy: creator.ID},
		{Name: "Org B", Description: "Desc B", CreatedBy: creator.ID},
		{Name: "Org C", Description: "Desc C", CreatedBy: creator.ID},
	}

	for _, org := range orgs {
		err := repo.Create(org)
		require.NoError(t, err)
	}

	// Test list
	foundOrgs, total, err := repo.List(10, 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, foundOrgs, 3)

	// Test with pagination
	foundOrgs, total, err = repo.List(2, 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, foundOrgs, 2)

	// Test with sorting
	foundOrgs, total, err = repo.List(10, 0, "name", "asc")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, foundOrgs, 3)
	assert.Equal(t, "Org A", foundOrgs[0].Name)
}

func TestOrganizationRepository_AddMember(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")
	user := createTestUser(t, db, "user@example.com", "Test User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Add member
	member := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           models.OrganizationRoleUser,
	}
	err = repo.AddMember(member)
	assert.NoError(t, err)

	// Test duplicate member
	duplicateMember := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           models.OrganizationRoleAdmin,
	}
	err = repo.AddMember(duplicateMember)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already a member")
}

func TestOrganizationRepository_GetMember(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization (creator is auto-added as admin)
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Get member
	member, err := repo.GetMember(org.ID, creator.ID)
	assert.NoError(t, err)
	assert.NotNil(t, member)
	assert.Equal(t, org.ID, member.OrganizationID)
	assert.Equal(t, creator.ID, member.UserID)
	assert.Equal(t, models.OrganizationRoleAdmin, member.Role)

	// Test non-existent member
	nonExistentUser := uuid.New()
	member, err = repo.GetMember(org.ID, nonExistentUser)
	assert.NoError(t, err)
	assert.Nil(t, member)
}

func TestOrganizationRepository_GetMembers(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")
	user1 := createTestUser(t, db, "user1@example.com", "User One")
	user2 := createTestUser(t, db, "user2@example.com", "User Two")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Add members
	member1 := &models.OrganizationMember{OrganizationID: org.ID, UserID: user1.ID, Role: models.OrganizationRoleUser}
	member2 := &models.OrganizationMember{OrganizationID: org.ID, UserID: user2.ID, Role: models.OrganizationRoleUser}

	err = repo.AddMember(member1)
	require.NoError(t, err)
	err = repo.AddMember(member2)
	require.NoError(t, err)

	// Get members (including creator)
	members, total, err := repo.GetMembers(org.ID, 10, 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total) // Creator + 2 added members
	assert.Len(t, members, 3)

	// Test pagination
	members, total, err = repo.GetMembers(org.ID, 2, 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, members, 2)
}

func TestOrganizationRepository_UpdateMember(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")
	user := createTestUser(t, db, "user@example.com", "Test User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Add member
	member := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           models.OrganizationRoleUser,
	}
	err = repo.AddMember(member)
	require.NoError(t, err)

	// Update member role
	member.Role = models.OrganizationRoleAdmin
	err = repo.UpdateMember(member)
	assert.NoError(t, err)

	// Verify update
	updatedMember, err := repo.GetMember(org.ID, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.OrganizationRoleAdmin, updatedMember.Role)
}

func TestOrganizationRepository_RemoveMember(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")
	user := createTestUser(t, db, "user@example.com", "Test User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Add member
	member := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           models.OrganizationRoleUser,
	}
	err = repo.AddMember(member)
	require.NoError(t, err)

	// Remove member
	err = repo.RemoveMember(org.ID, user.ID)
	assert.NoError(t, err)

	// Verify removal
	removedMember, err := repo.GetMember(org.ID, user.ID)
	assert.NoError(t, err)
	assert.Nil(t, removedMember)
}

func TestOrganizationRepository_CreateInvite(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Create invite
	invite := &models.OrganizationInvite{
		OrganizationID: org.ID,
		Email:          "invitee@example.com",
		Role:           models.OrganizationRoleUser,
		InvitedBy:      creator.ID,
		Status:         models.InviteStatusPending,
	}
	err = repo.CreateInvite(invite)
	assert.NoError(t, err)
	assert.NotEmpty(t, invite.Token)
	assert.True(t, invite.ExpiresAt.After(time.Now()))
}

func TestOrganizationRepository_GetInviteByToken(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Create invite
	invite := &models.OrganizationInvite{
		OrganizationID: org.ID,
		Email:          "invitee@example.com",
		Role:           models.OrganizationRoleUser,
		InvitedBy:      creator.ID,
		Status:         models.InviteStatusPending,
	}
	err = repo.CreateInvite(invite)
	require.NoError(t, err)

	// Get invite by token
	foundInvite, err := repo.GetInviteByToken(invite.Token)
	assert.NoError(t, err)
	assert.NotNil(t, foundInvite)
	assert.Equal(t, invite.ID, foundInvite.ID)
	assert.Equal(t, invite.Email, foundInvite.Email)

	// Test non-existent token
	foundInvite, err = repo.GetInviteByToken("non-existent-token")
	assert.NoError(t, err)
	assert.Nil(t, foundInvite)
}

func TestOrganizationRepository_GetPendingInviteByEmail(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Create invite
	invite := &models.OrganizationInvite{
		OrganizationID: org.ID,
		Email:          "invitee@example.com",
		Role:           models.OrganizationRoleUser,
		InvitedBy:      creator.ID,
		Status:         models.InviteStatusPending,
	}
	err = repo.CreateInvite(invite)
	require.NoError(t, err)

	// Get pending invite by email
	foundInvite, err := repo.GetPendingInviteByEmail(org.ID, "invitee@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, foundInvite)
	assert.Equal(t, invite.ID, foundInvite.ID)

	// Test non-existent email
	foundInvite, err = repo.GetPendingInviteByEmail(org.ID, "non-existent@example.com")
	assert.NoError(t, err)
	assert.Nil(t, foundInvite)
}

func TestOrganizationRepository_UpdateInvite(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Create invite
	invite := &models.OrganizationInvite{
		OrganizationID: org.ID,
		Email:          "invitee@example.com",
		Role:           models.OrganizationRoleUser,
		InvitedBy:      creator.ID,
		Status:         models.InviteStatusPending,
	}
	err = repo.CreateInvite(invite)
	require.NoError(t, err)

	// Update invite status
	now := time.Now()
	invite.Status = models.InviteStatusAccepted
	invite.AcceptedAt = &now
	err = repo.UpdateInvite(invite)
	assert.NoError(t, err)

	// Verify update
	updatedInvite, err := repo.GetInviteByID(invite.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.InviteStatusAccepted, updatedInvite.Status)
	assert.NotNil(t, updatedInvite.AcceptedAt)
}

func TestOrganizationRepository_ExpireInvites(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := NewOrganizationRepository(db)
	creator := createTestUser(t, db, "creator@example.com", "Creator User")

	// Create organization
	org := &models.Organization{
		Name:        "Test Organization",
		Description: "Test Description",
		CreatedBy:   creator.ID,
	}
	err := repo.Create(org)
	require.NoError(t, err)

	// Create expired invite by setting past expiry date
	invite := &models.OrganizationInvite{
		OrganizationID: org.ID,
		Email:          "invitee@example.com",
		Role:           models.OrganizationRoleUser,
		InvitedBy:      creator.ID,
		Status:         models.InviteStatusPending,
		ExpiresAt:      time.Now().Add(-24 * time.Hour), // Expired 1 day ago
	}
	// Create invite manually to set custom expiry
	invite.Token = "test-token"
	err = db.Create(invite).Error
	require.NoError(t, err)

	// Expire invites
	err = repo.ExpireInvites()
	assert.NoError(t, err)

	// Verify invite is expired
	expiredInvite, err := repo.GetInviteByID(invite.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.InviteStatusExpired, expiredInvite.Status)
}

func TestOrganizationRepository_BuildOrderClause(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := &OrganizationRepository{db: db}

	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		tableName string
		expected  string
	}{
		{"Valid org name asc", "name", "asc", "organizations", "name asc"},
		{"Valid org name desc", "name", "desc", "organizations", "name desc"},
		{"Invalid field", "invalid", "asc", "organizations", "created_at desc"},
		{"Invalid order", "name", "invalid", "organizations", "name desc"},
		{"CamelCase conversion", "createdAt", "asc", "organizations", "created_at asc"},
		{"Member table", "role", "desc", "organization_members", "role desc"},
		{"Invite table", "status", "asc", "organization_invites", "status asc"},
		{"Empty values", "", "", "organizations", "created_at desc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.buildOrderClause(tt.sortBy, tt.sortOrder, tt.tableName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOrganizationRepository_CamelToSnake(t *testing.T) {
	db, cleanup := setupOrgTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	repo := &OrganizationRepository{db: db}

	tests := []struct {
		input    string
		expected string
	}{
		{"createdAt", "created_at"},
		{"updatedAt", "updated_at"},
		{"firstName", "first_name"},
		{"name", "name"},
		{"ID", "i_d"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := repo.camelToSnake(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
