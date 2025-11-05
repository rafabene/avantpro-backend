package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/rafabene/avantpro-backend/internal/domain/entities"
	"github.com/rafabene/avantpro-backend/internal/domain/repositories"
	"github.com/rafabene/avantpro-backend/internal/domain/valueobjects"
)

// UserRepository implementa repositories.UserRepository
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository cria um novo UserRepository
func NewUserRepository(db *gorm.DB) repositories.UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *entities.User) error {
	model := r.toModel(user)

	db := r.getDB(ctx)
	if err := db.Create(model).Error; err != nil {
		return err
	}

	user.ID = model.ID
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*entities.User, error) {
	var model UserModel

	db := r.getDB(ctx)
	// Soft delete: ignorar registros deletados
	if err := db.Where("id = ? AND deleted_at IS NULL", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return r.toEntity(&model)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var model UserModel

	db := r.getDB(ctx)
	// Soft delete: ignorar registros deletados
	if err := db.Where("email = ? AND deleted_at IS NULL", email).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return r.toEntity(&model)
}

func (r *UserRepository) Update(ctx context.Context, user *entities.User) error {
	model := r.toModel(user)

	db := r.getDB(ctx)
	return db.Save(model).Error
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	db := r.getDB(ctx)
	// Soft delete: atualizar deleted_at ao invés de deletar
	now := time.Now().Unix()
	return db.Model(&UserModel{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", now).Error
}

func (r *UserRepository) List(ctx context.Context, filters repositories.UserFilters) ([]*entities.User, error) {
	var models []*UserModel

	db := r.getDB(ctx)
	query := db.Model(&UserModel{})

	// Soft delete: ignorar registros deletados
	query = query.Where("deleted_at IS NULL")

	// Aplicar filtros
	if filters.Role != nil {
		query = query.Where("role = ?", string(*filters.Role))
	}

	// Paginação
	page := filters.Page
	if page < 1 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	query = query.Limit(pageSize).Offset(offset)

	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	return r.toEntities(models)
}

// getDB extrai DB do contexto (para suportar transações)
func (r *UserRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

// Conversores
func (r *UserRepository) toModel(user *entities.User) *UserModel {
	var deletedAt *int64
	if user.DeletedAt != nil {
		ts := user.DeletedAt.Unix()
		deletedAt = &ts
	}

	return &UserModel{
		ID:           user.ID,
		Email:        user.Email.String(),
		Name:         user.Name,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
		AvatarURL:    user.AvatarURL,
		CreatedAt:    user.CreatedAt.Unix(),
		UpdatedAt:    user.UpdatedAt.Unix(),
		DeletedAt:    deletedAt,
	}
}

func (r *UserRepository) toEntity(model *UserModel) (*entities.User, error) {
	email, err := valueobjects.NewEmail(model.Email)
	if err != nil {
		return nil, err
	}

	var deletedAt *time.Time
	if model.DeletedAt != nil {
		ts := time.Unix(*model.DeletedAt, 0)
		deletedAt = &ts
	}

	return &entities.User{
		ID:           model.ID,
		Email:        email,
		Name:         model.Name,
		PasswordHash: model.PasswordHash,
		Role:         entities.Role(model.Role),
		AvatarURL:    model.AvatarURL,
		CreatedAt:    time.Unix(model.CreatedAt, 0),
		UpdatedAt:    time.Unix(model.UpdatedAt, 0),
		DeletedAt:    deletedAt,
	}, nil
}

func (r *UserRepository) toEntities(models []*UserModel) ([]*entities.User, error) {
	entities := make([]*entities.User, 0, len(models))

	for _, model := range models {
		entity, err := r.toEntity(model)
		if err != nil {
			return nil, err
		}
		entities = append(entities, entity)
	}

	return entities, nil
}
