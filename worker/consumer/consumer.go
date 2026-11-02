package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"files-uploader/internal/events"
	"files-uploader/internal/service"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	queueKey      = "minio:events"
	processingKey = "minio:events:processing"
)

type Consumer struct {
	rdb     *redis.Client
	service service.FileService
}

func NewConsumer(rdb *redis.Client, service service.FileService) *Consumer {
	return &Consumer{rdb, service}
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

func (w *Consumer) Run(ctx context.Context) error {
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
		w.rdb.LRem(ctx, processingKey, 1, raw)
	}
}

func (w *Consumer) handle(ctx context.Context, raw string) error {
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
			file, err := w.service.ProcessUpload(ctx, key)
			if err != nil {
				return err
			}
			if file == nil {
				continue
			}

			ev := events.UploadedEvent{
				UserID:      file.UserID.String(),
				FileID:      file.ID.String(),
				ObjectKey:   file.ObjectKey,
				Name:        file.FileName,
				Size:        file.SizeBites,
				ContentType: file.ContentType,
				CreatedAt:   file.CreatedAt,
			}
			if err := events.PublishUploaded(ctx, w.rdb, ev); err != nil {
				slog.Error("unable to publish upload event", slog.Any("error", err))
			}
		}
	}
	return nil
}
