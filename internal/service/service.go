package service

import (
	"context"
	"time"

	"files-uploader/internal/models"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type FileService interface {
	ProcessUpload(ctx context.Context, key string) error
	SaveFileMetadata(ctx context.Context, file *models.File) error
	PresignSingle(ctx context.Context, key, contentType string, size int) (*v4.PresignedHTTPRequest, time.Time, error)
	PresignMultipart(ctx context.Context, file *models.File) (*v4.PresignedHTTPRequest, time.Time, error)
	FindFileByMultipartID(ctx context.Context, id string) (*models.File, error)
	PresignPartRange(ctx context.Context, key, s3UploadID string, nums []int) ([]*v4.PresignedHTTPRequest, time.Time, error)
	ListAllParts(ctx context.Context, key, uploadID string) (*s3.ListPartsOutput, error)
	CompleteFileUpload(context.Context, *models.File, *s3.ListPartsOutput) error
	ListFiles(context.Context, uuid.UUID) ([]*models.File, error)
	PresignDownload(ctx context.Context, key, fileName, contentType string) (*v4.PresignedHTTPRequest, time.Time, error)
}
