package repository

import (
	"context"

	"files-uploader/internal/models"

	"github.com/google/uuid"
)

type FilesRepository interface {
	Save(context.Context, *models.File) error
	FindByKey(ctx context.Context, key string) (*models.File, error)
	Delete()
	Update(cxt context.Context, f *models.File) error
	FindByUploadID(ctx context.Context, uploadID string) (*models.File, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.File, error)
}

type UserRepository interface {
	Save(context.Context, *models.User) error
	GetByEmail(context.Context, string) (*models.User, error)
	GetByID(context.Context, uuid.UUID) (*models.User, error)
	ExistForEmail(context.Context, string) (bool, error)
}
