package service

import (
	"context"
	"files-uploader/internal/models"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

type FileService interface {
	ProcessUpload(ctx context.Context, key string) error
	SaveFileMetadata(ctx context.Context, file *models.File) error
	PresignSingle(ctx context.Context, key, contentType string, size int) (*v4.PresignedHTTPRequest, time.Time, error)
	PresignMultipart(ctx context.Context, file *models.File) (*v4.PresignedHTTPRequest, time.Time, error)
	FindFileByMultipartID(ctx context.Context, id string) (*models.File, error)
	PresignPartRange(ctx context.Context, key, s3UploadID string, nums []int) ([]*v4.PresignedHTTPRequest, time.Time, error)
}
