package worker

import (
	"context"
	"encoding/json"
	"errors"
	"files-uploader/internal/models"
	"files-uploader/internal/repository"
	"files-uploader/internal/storage"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
)

const (
	queueKey      = "minio:events"
	processingKey = "minio:events:processing"
	MaxSize       = 1 << 32 // 4,
)

type Worker struct {
	rdb    *redis.Client
	repo   repository.FilesRepository
	client *s3.Client
}

func NewUploadWorker(rdb *redis.Client, repo repository.FilesRepository, client *s3.Client) *Worker {
	return &Worker{rdb, repo, client}
}

type minioEntry struct {
	Event []struct {
		S3 struct {
			Bucket struct{ Name string } `json:"bucket"`
			Object struct {
				Key  string `json:"key"`
				Size int64  `json:"size"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Event"`
}

func (w *Worker) Run(ctx context.Context) error {
	slog.Info("running worker")
	for {
		raw, err := w.rdb.BLMove(ctx, queueKey, processingKey, "LEFT", "RIGHT", 5*time.Second).Result()
		if errors.Is(err, redis.Nil) {
			continue // timeout, no events
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			time.Sleep(time.Second)
			continue
		}

		if err := w.handle(ctx, raw); err != nil {
			slog.Error("event failed, left in processing list", slog.Any("error", err))
			continue
		}
		slog.Info("file successfully updated")
		w.rdb.LRem(ctx, processingKey, 1, raw)
	}
}

func (w *Worker) handle(ctx context.Context, raw string) error {
	var entries []minioEntry // try a single object if your version emits one
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return fmt.Errorf("bad event: %w", err)
	}
	for _, e := range entries {
		for _, ev := range e.Event {
			key, err := url.QueryUnescape(ev.S3.Object.Key) // keys come URL-encoded
			if err != nil {
				return err
			}
			if err := w.processUpload(ctx, key); err != nil {
				return err
			}
		}
	}
	return nil
}
func (w *Worker) processUpload(ctx context.Context, key string) error {

	// first find the file in db
	file, err := w.repo.FindByKey(ctx, key)
	if err != nil {
		return err
	}
	if file.Status != string(models.PendingStatus) {
		return errors.New("file is already processed")
	}

	head, err := w.client.HeadObject(
		ctx,
		&s3.HeadObjectInput{Bucket: aws.String(storage.Bucket), Key: aws.String(key)},
	)
	if err != nil {
		return err
	}

	// validate that size in s3 and db are the ame
	if *head.ContentLength != int64(file.SizeBites) || *head.ContentLength > MaxSize {
		// return w.reject(ctx, up, key, "type mismatch")
		return nil
	}

	//TODO: analize file for malware

	// promote file, set as upload
	// finalKey := fmt.Sprintf("files/%s/%s", file.UserID, file.ID)
	// set as updated and update the obj key
	file.SetAsUploaded()
	file.GenerateFinalKey()
	_, err = w.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(storage.Bucket),
		Key:        aws.String(file.ObjectKey),
		CopySource: aws.String(url.PathEscape(storage.Bucket + "/" + key)),
	})
	if err != nil {
		return err
	}

	if err := w.repo.Update(ctx, file); err != nil {
		return err
	}
	_, err = w.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(storage.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	return nil
}
