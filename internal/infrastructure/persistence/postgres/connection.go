package postgres

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rafabene/avantpro-backend/internal/domain/ports"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/config"
)

// NewDatabaseConnection cria uma nova conexão com o PostgreSQL
func NewDatabaseConnection(cfg *config.DatabaseConfig, log ports.Logger) (*gorm.DB, error) {
	// GORM config
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: false,
	}

	// Conectar
	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configurar connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConns)
	sqlDB.SetMaxIdleConns(cfg.MinConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxIdleTime) * time.Second)

	// Ping para verificar conexão
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connected successfully",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.DBName,
	)

	return db, nil
}
