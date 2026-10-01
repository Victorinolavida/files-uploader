package service

import (
	"context"
	"errors"
	"files-uploader/config"
	"files-uploader/internal/models"
	"files-uploader/internal/repository"
	"log/slog"
	"net/url"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var validStatus = []string{
	string(models.PendingStatus), string(models.Validating),
}

const MaxSize = 1 << 32 // 4,

type FileServ struct {
	repo          repository.FilesRepository
	client        *s3.Client
	bucket        string
	presignClient *s3.PresignClient
}

func NewFileServices(repo repository.FilesRepository, client *s3.Client, cfg config.BucketConfig) *FileServ {
	fileServ := &FileServ{
		repo:          repo,
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.Bucket,
	}
	return fileServ
}

var _ FileService = (*FileServ)(nil)

func (serv *FileServ) ProcessUpload(ctx context.Context, key string) error {
	file, err := serv.repo.FindByKey(ctx, key)
	if err != nil {
		return err
	}
	if !slices.Contains(validStatus, file.Status) {
		return errors.New("file is already processed or rejected")
	}

	head, err := serv.client.HeadObject(
		ctx,
		&s3.HeadObjectInput{Bucket: aws.String(serv.bucket), Key: aws.String(key)},
	)
	if err != nil {
		return err
	}

	// validate that size in s3 and db are the ame
	if *head.ContentLength != int64(file.SizeBites) || *head.ContentLength > MaxSize {
		file.Reject("size mistmatch")
		return serv.reject(ctx, file)
	}

	//TODO: analize file for malware

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
		return serv.reject(ctx, file)
	}

	if err := serv.repo.Update(ctx, file); err != nil {
		return err
	}
	_, err = serv.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(serv.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	slog.Info("file successfully updated")
	return nil
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
