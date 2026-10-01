package main

import (
	"context"
	"files-uploader/config"
	"files-uploader/internal/db"
	"files-uploader/internal/repository"
	"files-uploader/internal/service"
	"files-uploader/internal/storage"
	"files-uploader/pkg/logger"
	"files-uploader/worker/consumer"
	"log"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

func main() {

	appConfig, err := config.Load()
	if err != nil {
		log.Panicf("error loading config")
	}

	log := logger.NewLogger(appConfig.AppEnv)
	cfg, err := storage.LoadConfig(context.Background(), appConfig.AppEnv, appConfig.BucketConfig)

	if err != nil {
		log.Error("unable to load SDK config", slog.Any("error", err))
		panic(1)
	}

	objStorage, err := storage.NewObjectStorate(context.TODO(), cfg, appConfig.BucketConfig)
	if err != nil {
		log.Error("unable to connect to bucket", slog.Any("error", err))
		panic(1)
	}
	log.Info("connected to bucket")

	db := db.NewDB()
	repo := repository.NewSqlRepo(db)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	stausCmd := redisClient.Ping(context.Background())
	if err := stausCmd.Err(); err != nil {
		log.Error("unable to connect redis", slog.Any("error", err))
	}
	slog.Info("connected to redis")

	fileService := service.NewFileServices(repo, objStorage, appConfig.BucketConfig)
	redisConsumer := consumer.NewConsumer(redisClient, fileService)
	if err := redisConsumer.Run(context.Background()); err != nil {
		slog.Error("running worker", slog.Any("error", err))
		panic(1)
	}
}
