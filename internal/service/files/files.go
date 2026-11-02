package fileservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"time"

	"files-uploader/config"
	"files-uploader/internal/models"
	"files-uploader/internal/repository"
	"files-uploader/internal/service"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

var validStatus = []string{
	string(models.PendingStatus), string(models.Validating),
}

var validStatusMultipart = []string{
	string(models.PendingStatus), string(models.Validating), string(models.UploadingStatus),
}

type FileServ struct {
	repo          repository.FilesRepository
	client        *s3.Client
	bucket        string
	presignClient *s3.PresignClient
}

func NewFileServices(repo repository.FilesRepository, client *s3.Client, cfg config.BucketConfig) service.FileService {
	fileServ := &FileServ{
		repo:          repo,
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.Bucket,
	}
	return fileServ
}

func (serv *FileServ) ProcessUpload(ctx context.Context, key string) (*models.File, error) {
	file, err := serv.repo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if file.Mode == string(models.SinglePart) && !slices.Contains(validStatus, file.Status) {
		return nil, errors.New("file is already processed or rejected")
	}

	if file.Mode == string(models.MultiPart) && !slices.Contains(validStatusMultipart, file.Status) {
		return nil, errors.New("file is already processed or rejected")
	}

	head, err := serv.client.HeadObject(
		ctx,
		&s3.HeadObjectInput{Bucket: aws.String(serv.bucket), Key: aws.String(key)},
	)
	if err != nil {
		return nil, err
	}

	// validate that size in s3 and db are the ame
	if *head.ContentLength != int64(file.SizeBites) || *head.ContentLength > models.MaxSize {
		file.Reject("size mistmatch")
		return nil, serv.reject(ctx, file)
	}

	// TODO: analize file for malware

	// promote file, set as upload
	// set as updated and update the obj key
	file.SetAsUploaded()
	_, err = serv.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(serv.bucket),
		Key:        aws.String(file.ObjectKey),
		CopySource: aws.String(url.PathEscape(serv.bucket + "/" + key)),
	})
	if err != nil {
		file.Reject("unable to upload file")
		return nil, serv.reject(ctx, file)
	}

	if err := serv.repo.Update(ctx, file); err != nil {
		return nil, err
	}
	_, err = serv.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(serv.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		slog.Error("unable to delete quarantine object", slog.Any("error", err))
	}

	slog.Info("file successfully updated")
	return file, nil
}

func (serv *FileServ) reject(ctx context.Context, file *models.File) error {
	slog.Info("rejecting file")
	// first mark the file as rejected
	if err := serv.repo.Update(ctx, file); err != nil {
		return err
	}

	// then delete the file from s3
	_, err := serv.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(serv.bucket),
		Key:    aws.String(file.ObjectKey),
	})
	if err != nil {
		return err
	}

	slog.Info("rejecting file sucessfully")
	return nil
}

func (serv *FileServ) SaveFileMetadata(ctx context.Context, file *models.File) error {
	err := serv.repo.Save(ctx, file)
	if err != nil {
		return err
	}
	return nil
}

func (serv *FileServ) PresignMultipart(ctx context.Context, file *models.File) (*v4.PresignedHTTPRequest, time.Time, error) {
	ttl := 1 * time.Hour
	out, err := serv.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(serv.bucket),
		Key:         aws.String(file.ObjectKey),
		ContentType: aws.String(file.ContentType),
	})
	if err != nil {
		return nil, time.Now().Add(ttl), err
	}
	if out.UploadId == nil {
		return nil, time.Now().Add(ttl), fmt.Errorf("S3 did not return an upload ID")
	}
	file.MultipartUploadID = out.UploadId
	if err = serv.repo.Update(ctx, file); err != nil {
		slog.Error("can not update multipart_upload_id", slog.Any("eror", err))
		return nil, time.Now().Add(ttl), err
	}
	request, err := serv.presignClient.PresignUploadPart(
		ctx,
		&s3.UploadPartInput{
			Bucket:     aws.String(serv.bucket),
			Key:        aws.String(file.ObjectKey),
			UploadId:   aws.String(*file.MultipartUploadID),
			PartNumber: aws.Int32(int32(1)),
		},
		func(po *s3.PresignOptions) {
			po.Expires = ttl
		},
	)
	return request, time.Now().Add(ttl), err
}

func (serv *FileServ) PresignSingle(ctx context.Context, key, contentType string, size int) (*v4.PresignedHTTPRequest, time.Time, error) {
	ttl := 15 * time.Minute
	req, err := serv.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(serv.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(size)),
	}, s3.WithPresignExpires(ttl))
	return req, time.Now().Add(ttl), err
}

func (serv *FileServ) FindFileByMultipartID(ctx context.Context, id string) (*models.File, error) {
	return serv.repo.FindByUploadID(ctx, id)
}

func (serv *FileServ) PresignPartRange(ctx context.Context, key, s3UploadID string, nums []int) ([]*v4.PresignedHTTPRequest, time.Time, error) {
	ttl := 1 * time.Hour
	requests := make([]*v4.PresignedHTTPRequest, len(nums))
	for idx, i := range nums {
		request, err := serv.presignClient.PresignUploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(serv.bucket),
			Key:        aws.String(key),
			UploadId:   aws.String(s3UploadID),
			PartNumber: aws.Int32(int32(i)),
		}, s3.WithPresignExpires(ttl))
		if err != nil {
			return nil, time.Now(), err
		}
		requests[idx] = request

	}

	return requests, time.Now().Add(ttl), nil
}

func (serv *FileServ) ListAllParts(ctx context.Context, key, uploadID string) (*s3.ListPartsOutput, error) {
	return serv.client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(serv.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
}

func (serv *FileServ) CompleteFileUpload(ctx context.Context, file *models.File, parst *s3.ListPartsOutput) error {
	if err := file.SetAsUploading(); err != nil {
		return err
	}
	if err := serv.repo.Update(ctx, file); err != nil {
		return err
	}

	completed := make([]types.CompletedPart, 0, len(parst.Parts))
	for _, p := range parst.Parts {
		completed = append(completed, types.CompletedPart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}
	_, err := serv.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(serv.bucket),
		Key:             aws.String(file.ObjectKey),
		UploadId:        aws.String(*file.MultipartUploadID),
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completed},
	})

	return err
}

func (serv *FileServ) ListFiles(ctx context.Context, userID uuid.UUID) ([]*models.File, error) {
	return serv.repo.ListByUser(ctx, userID)
}

func (serv *FileServ) PresignDownload(ctx context.Context, key, fileName, contentType string) (*v4.PresignedHTTPRequest, time.Time, error) {
	ttl := 15 * time.Minute
	req, err := serv.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(serv.bucket),
		Key:                        aws.String(key),
		ResponseContentType:        aws.String(contentType),
		ResponseContentDisposition: aws.String(fmt.Sprintf("attachment; filename=%q", fileName)),
	}, s3.WithPresignExpires(ttl))
	return req, time.Now().Add(ttl), err
}
