package userrepository

import (
	"context"
	"database/sql"
	"errors"

	"files-uploader/internal/models"
	"files-uploader/internal/repository"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UserRepo struct {
	db *bun.DB
}

var ErrUserNotFound = errors.New("user not found")

func NewUserRepo(db *bun.DB) repository.UserRepository {
	return &UserRepo{db}
}

func (repo *UserRepo) Save(ctx context.Context, u *models.User) error {
	_, err := repo.db.NewInsert().Model(u).Exec(ctx)
	return err
}

func (repo *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	u := new(models.User)
	err := repo.db.NewSelect().Model(u).Where("email = ?", email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func (repo *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u := new(models.User)
	err := repo.db.NewSelect().Model(u).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func (repo *UserRepo) ExistForEmail(ctx context.Context, email string) (bool, error) {
	u := new(models.User)
	return repo.db.NewSelect().Model(u).Where("email = ?", email).Exists(ctx)
}
