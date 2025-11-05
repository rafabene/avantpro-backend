package repositories

import (
	"context"

	"github.com/rafabene/avantpro-backend/internal/domain/entities"
)

// UserRepository define a interface para persistência de usuários
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	FindByID(ctx context.Context, id string) (*entities.User, error)
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters UserFilters) ([]*entities.User, error)
}

// UserFilters contém filtros para listagem de usuários
type UserFilters struct {
	Role     *entities.Role
	Page     int // Página (começa em 1)
	PageSize int // Itens por página (default: 20, max: 100)
}
