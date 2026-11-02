package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"text/template"

	"files-uploader/config"
	"files-uploader/internal/db"
	appRouter "files-uploader/internal/http/files"
	filesrepository "files-uploader/internal/repository/files"
	fileservice "files-uploader/internal/service/files"
	"files-uploader/internal/storage"
	"files-uploader/pkg/logger"
)

func render(w http.ResponseWriter, data any) {
	tpl := template.Must(template.ParseFiles(
		"views/layouts/base.html",
	))
	if err := tpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

	db := db.NewDB()
	repo := filesrepository.NewFilesRepository(db)
	mux := http.NewServeMux()

	fileService := fileservice.NewFileServices(repo, objStorage, appConfig.BucketConfig)
	handler := appRouter.NewRouter(render, fileService)
	mux.Handle("/", handler)

	log.Info("starting server", slog.Int("port", appConfig.Port))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", appConfig.Port), mux); err != nil {
		log.Error("error starting server", slog.Any("error", err))
		panic(1)
	}
}
