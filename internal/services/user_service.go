package services

import (
	"context"

	"github.com/rafabene/avantpro-backend/internal/domain/entities"
	"github.com/rafabene/avantpro-backend/internal/domain/errors"
	"github.com/rafabene/avantpro-backend/internal/domain/ports"
	"github.com/rafabene/avantpro-backend/internal/domain/repositories"
)

// UserService contém a lógica de negócio para usuários
type UserService struct {
	userRepo repositories.UserRepository
	uow      ports.UnitOfWork
	logger   ports.Logger
}

// NewUserService cria um novo UserService
func NewUserService(
	userRepo repositories.UserRepository,
	uow ports.UnitOfWork,
	logger ports.Logger,
) *UserService {
	return &UserService{
		userRepo: userRepo,
		uow:      uow,
		logger:   logger,
	}
}

// CreateUserInput representa os dados para criar um usuário
type CreateUserInput struct {
	Email    string
	Name     string
	Password string
}

// CreateUser cria um novo usuário
func (s *UserService) CreateUser(ctx context.Context, input CreateUserInput) (*entities.User, error) {
	s.logger.Info("creating user", "email", input.Email)

	// Validar se email já existe
	existing, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.ErrEmailAlreadyExists
	}

	// TODO: Implementar criação de usuário com hash de senha
	// Por enquanto, apenas estrutura base
	s.logger.Info("user creation logic not fully implemented yet")

	return nil, nil
}

// GetUser busca um usuário por ID
func (s *UserService) GetUser(ctx context.Context, id string) (*entities.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.ErrUserNotFound
	}
	return user, nil
}

// ListUsers lista usuários com filtros
func (s *UserService) ListUsers(ctx context.Context, filters repositories.UserFilters) ([]*entities.User, error) {
	return s.userRepo.List(ctx, filters)
}
