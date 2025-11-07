package domain

import "context"

// UnitOfWork define a interface para gerenciamento de transações
type UnitOfWork interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	WithTransaction(ctx context.Context, fn func(context.Context) error) error
}
