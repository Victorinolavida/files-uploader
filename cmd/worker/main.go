package main

import (
	"context"
	"files-uploader/internal/db"
	"files-uploader/internal/repository"
	"files-uploader/internal/storage"
	"files-uploader/worker"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/redis/go-redis/v9"
)

func main() {

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(log)

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	stausCmd := client.Ping(context.Background())
	if err := stausCmd.Err(); err != nil {
		slog.Error("unable to connect redis", slog.Any("error", err))
	}
	slog.Info("connected to redis")
	db := db.NewDB()

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				"minioadmin",
				"minioadmin",
				"",
			),
		))

	if err != nil {
		slog.Error("unable to load SDK config", slog.Any("error", err))
		panic(1)
	}

	objStorage, err := storage.NewObjectStorate(context.TODO(), cfg)
	if err != nil {
		slog.Error("unable to connect to bucket", slog.Any("error", err))
		panic(1)
	}

	slog.Info("connected to object storage")

	fileRepo := repository.NewSqlRepo(db)
	worker := worker.NewUploadWorker(client, fileRepo, objStorage)
	if err := worker.Run(context.Background()); err != nil {
		slog.Error("running worker", slog.Any("error", err))
		panic(1)
	}
}
