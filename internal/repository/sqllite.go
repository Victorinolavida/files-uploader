package repository

import (
	"context"
	"files-uploader/internal/models"

	"github.com/uptrace/bun"
)

type sqlLiteRepo struct {
	db *bun.DB
}

var _ FilesRepository = (*sqlLiteRepo)(nil)

func NewSqlRepo(db *bun.DB) *sqlLiteRepo {
	return &sqlLiteRepo{db}
}

func (repo *sqlLiteRepo) Save(ctx context.Context, f *models.File) error {
	_, err := repo.db.NewInsert().Model(f).Exec(ctx)
	return err
}
func (repo *sqlLiteRepo) Delete() {}

func (repo *sqlLiteRepo) Update(cxt context.Context, f *models.File) error {
	_, err := repo.db.NewUpdate().Model(f).WherePK().Exec(cxt)
	return err
}

func (repo *sqlLiteRepo) FindByKey(ctx context.Context, key string) (*models.File, error) {
	file := new(models.File)
	err := repo.db.NewSelect().Model(file).Where("object_key = ?", key).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (repo *sqlLiteRepo) FindByUploadID(ctx context.Context, uploadID string) (*models.File, error) {
	file := new(models.File)
	err := repo.db.NewSelect().Model(file).Where(" multipart_upload_id = ?", uploadID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return file, nil
}
