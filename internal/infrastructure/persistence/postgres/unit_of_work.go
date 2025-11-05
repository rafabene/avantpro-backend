package postgres

import (
	"context"

	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/domain/ports"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const txKey contextKey = "tx"

// UnitOfWork implementa ports.UnitOfWork
type UnitOfWork struct {
	db *gorm.DB
}

// NewUnitOfWork cria um novo UnitOfWork
func NewUnitOfWork(db *gorm.DB) ports.UnitOfWork {
	return &UnitOfWork{db: db}
}

func (uow *UnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	tx := uow.db.Begin()
	return context.WithValue(ctx, txKey, tx), nil
}

func (uow *UnitOfWork) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	if !ok {
		return nil
	}
	return tx.Commit().Error
}

func (uow *UnitOfWork) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	if !ok {
		return nil
	}
	return tx.Rollback().Error
}

func (uow *UnitOfWork) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx := uow.db.Begin()

	txCtx := context.WithValue(ctx, txKey, tx)

	err := fn(txCtx)

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
