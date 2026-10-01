package repository

import (
	"context"
	"files-uploader/internal/models"
)

type FilesRepository interface {
	Save(context.Context, *models.File) error
	FindByKey(ctx context.Context, key string) (*models.File, error)
	Delete()
	Update(cxt context.Context, f *models.File) error
}
