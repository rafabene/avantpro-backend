package database

import (
	"fmt"

	"gorm.io/gorm"
)

// MigrateToUUID migrates tables from integer ID to UUID
func MigrateToUUID(db *gorm.DB) error {
	// Check if the users table exists and has integer ID
	var hasUsersTable bool
	err := db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&hasUsersTable).Error
	if err != nil {
		return fmt.Errorf("failed to check if users table exists: %w", err)
	}

	if !hasUsersTable {
		// Table doesn't exist yet, normal migration will handle it
		return enableUUIDExtension(db)
	}

	// Check current ID column type
	var columnType string
	err = db.Raw(`
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_name = 'users' AND column_name = 'id'
	`).Scan(&columnType).Error
	if err != nil {
		return fmt.Errorf("failed to check ID column type: %w", err)
	}

	// If already UUID, nothing to do
	if columnType == "uuid" {
		return nil
	}

	// Begin transaction for migration
	return db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Drop all data (since we're in development)
		if err := tx.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error; err != nil {
			return fmt.Errorf("failed to truncate users table: %w", err)
		}

		if err := tx.Exec("TRUNCATE TABLE profiles RESTART IDENTITY CASCADE").Error; err != nil {
			// Ignore error if profiles table doesn't exist
			_ = err
		}

		// Step 2: Drop the old tables completely
		if err := tx.Exec("DROP TABLE IF EXISTS profiles CASCADE").Error; err != nil {
			return fmt.Errorf("failed to drop profiles table: %w", err)
		}

		if err := tx.Exec("DROP TABLE IF EXISTS users CASCADE").Error; err != nil {
			return fmt.Errorf("failed to drop users table: %w", err)
		}

		// Step 3: Enable UUID extension
		return enableUUIDExtension(tx)
	})
}

func enableUUIDExtension(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}
	return nil
}
