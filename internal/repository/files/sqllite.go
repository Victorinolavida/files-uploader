package filesrepository

import (
	"context"

	"files-uploader/internal/models"
	"files-uploader/internal/repository"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type sqlLiteFilesRepo struct {
	db *bun.DB
}

func NewFilesRepository(db *bun.DB) repository.FilesRepository {
	return &sqlLiteFilesRepo{db}
}

func (repo *sqlLiteFilesRepo) Save(ctx context.Context, f *models.File) error {
	_, err := repo.db.NewInsert().Model(f).Exec(ctx)
	return err
}
func (repo *sqlLiteFilesRepo) Delete() {}

func (repo *sqlLiteFilesRepo) Update(cxt context.Context, f *models.File) error {
	_, err := repo.db.NewUpdate().Model(f).WherePK().Exec(cxt)
	return err
}

func (repo *sqlLiteFilesRepo) FindByKey(ctx context.Context, key string) (*models.File, error) {
	file := new(models.File)
	err := repo.db.NewSelect().Model(file).Where("object_key = ?", key).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (repo *sqlLiteFilesRepo) FindByUploadID(ctx context.Context, uploadID string) (*models.File, error) {
	file := new(models.File)
	err := repo.db.NewSelect().Model(file).Where(" multipart_upload_id = ?", uploadID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (repo *sqlLiteFilesRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.File, error) {
	files := make([]*models.File, 0)
	err := repo.db.NewSelect().
		Model(&files).
		Where("user_id = ?", userID).
		Where("status = ?", string(models.Uploaded)).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return files, nil
}
